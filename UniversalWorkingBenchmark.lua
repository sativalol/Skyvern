local ITER = 100000
local HEAT = 500

local res = {}
local ord = {}

local function bench(tag, fn)
	table.insert(ord, tag)
	for i = 1, HEAT do fn(i) end
	local t = tick()
	for i = 1, ITER do fn(i) end
	res[tag] = tick() - t
end

local function rpad(s, n)
	s = tostring(s)
	while #s < n do s = s .. " " end
	return s
end

local function fmt(n)
	return string.format("%.3f ms", n * 1000)
end

-- -----------------------------------------------
-- 1. basic arithmetic (how bad does obf mangle ops)
-- -----------------------------------------------
bench("arithmetic", function(i)
	local x = i * 3.14159
	local y = x / (i + 1)
	local _ = y - i % 7 + i ^ 0.5
end)

-- -----------------------------------------------
-- 2. lots of locals (register pressure, renaming cost)
-- -----------------------------------------------
bench("many locals", function(i)
	local a = i + 1
	local b = i + 2
	local c = i + 3
	local d = i + 4
	local e = i + 5
	local f = a + b + c
	local g = d + e + f
	local _ = g * 2 - a
end)

-- -----------------------------------------------
-- 3. upvalue capture (closures stress obf badly)
-- -----------------------------------------------
bench("upvalue closure", function(i)
	local x = i * 2
	local y = i + 7
	local function inner()
		return x + y
	end
	local _ = inner()
end)

-- -----------------------------------------------
-- 4. chained closures / returned functions
-- -----------------------------------------------
bench("closure chain", function(i)
	local function make(n)
		return function() return n * n end
	end
	local _ = make(i)()
end)

-- -----------------------------------------------
-- 5. string building (concat overhead, tostring cost)
-- -----------------------------------------------
bench("string concat", function(i)
	local _ = "item_" .. tostring(i) .. "_end"
end)

-- -----------------------------------------------
-- 6. string.format (vararg, format parsing)
-- -----------------------------------------------
bench("string.format", function(i)
	local _ = string.format("%05d | %.4f", i, i * 0.001)
end)

-- -----------------------------------------------
-- 7. string methods (sub, rep, len)
-- -----------------------------------------------
bench("string methods", function(i)
	local s = tostring(i)
	local _ = string.sub(s, 1, 2)
	local __ = string.rep(s, 3)
	local ___ = #s
end)

-- -----------------------------------------------
-- 8. pattern match (common in real code)
-- -----------------------------------------------
bench("pattern match", function(i)
	local s = "key=" .. tostring(i) .. ";flag=true"
	local _ = string.match(s, "key=(%d+)")
end)

-- -----------------------------------------------
-- 9. table read/write (array part)
-- -----------------------------------------------
local _tbl = {}
bench("table array r/w", function(i)
	_tbl[i % 128 + 1] = i
	local _ = _tbl[(i - 1) % 128 + 1]
end)

-- -----------------------------------------------
-- 10. table hash part (string keys, common in OOP)
-- -----------------------------------------------
bench("table hash r/w", function(i)
	local t = { x = i, y = i + 1, z = i - 1 }
	t.w = t.x + t.y
	local _ = t.w - t.z
end)

-- -----------------------------------------------
-- 11. table.insert / remove (obf can't optimize this)
-- -----------------------------------------------
local _q = {}
bench("table insert/remove", function(i)
	table.insert(_q, i)
	if #_q > 64 then table.remove(_q, 1) end
end)

-- -----------------------------------------------
-- 12. metatables / __index (very common in real modules)
-- -----------------------------------------------
local _base = { mul = function(self, n) return self.val * n end }
_base.__index = _base
bench("metatable __index", function(i)
	local obj = setmetatable({ val = i }, _base)
	local _ = obj:mul(3)
end)

-- -----------------------------------------------
-- 13. metatable __index as function (harder for obf)
-- -----------------------------------------------
bench("__index function", function(i)
	local obj = setmetatable({}, {
		__index = function(_, k) return k .. tostring(i) end
	})
	local _ = obj.foo
end)

-- -----------------------------------------------
-- 14. pcall happy path
-- -----------------------------------------------
bench("pcall (no error)", function(i)
	local ok, v = pcall(function() return i * 2 end)
	local _ = ok and v
end)

-- -----------------------------------------------
-- 15. pcall error path (error() is slow, tests obf wrapping)
-- -----------------------------------------------
bench("pcall (errors)", function(i)
	pcall(function()
		if i % 3 == 0 then error("bad") end
	end)
end)

-- -----------------------------------------------
-- 16. math lib (global lookup each call vs localized)
-- -----------------------------------------------
local _floor = math.floor
local _sqrt  = math.sqrt
local _sin   = math.sin
bench("math (localized)", function(i)
	local _ = _floor(_sqrt(i + 1) * 100) + _floor(_sin(i) * 10)
end)

bench("math (global ref)", function(i)
	local _ = math.floor(math.sqrt(i + 1) * 100) + math.floor(math.sin(i) * 10)
end)

-- -----------------------------------------------
-- 17. boolean branching (branch prediction + obf conditionals)
-- -----------------------------------------------
bench("boolean branches", function(i)
	local a = i > 0 and i < ITER
	local b = i % 2 == 0 or i % 3 == 0
	local c = not (a and b)
	local _ = c or a
end)

-- -----------------------------------------------
-- 18. numeric for with inner work (loop overhead)
-- -----------------------------------------------
bench("inner numeric for", function(i)
	local sum = 0
	for j = 1, 10 do
		sum = sum + j * i
	end
	local _ = sum
end)

-- -----------------------------------------------
-- 19. ipairs iteration (obf sometimes breaks iterators)
-- -----------------------------------------------
local _arr = {10, 20, 30, 40, 50, 60, 70, 80}
bench("ipairs", function(_)
	local sum = 0
	for _, v in ipairs(_arr) do
		sum = sum + v
	end
	local __ = sum
end)

-- -----------------------------------------------
-- 20. pairs iteration (hash iter, order undefined)
-- -----------------------------------------------
local _map = { a=1, b=2, c=3, d=4, e=5, f=6 }
bench("pairs", function(_)
	local sum = 0
	for _, v in pairs(_map) do
		sum = sum + v
	end
	local __ = sum
end)

-- -----------------------------------------------
-- 21. varargs (... forwarding is a known obf pain point)
-- -----------------------------------------------
local function _sum(...)
	local t = {...}
	local n = 0
	for i = 1, #t do n = n + t[i] end
	return n
end
bench("varargs (...)", function(i)
	local _ = _sum(i, i+1, i+2, i+3, i+4)
end)

-- -----------------------------------------------
-- 22. select with varargs
-- -----------------------------------------------
local function _sel(...)
	return select("#", ...), select(2, ...)
end
bench("select + varargs", function(i)
	local n, _ = _sel(i, i*2, i*3)
	local __ = n
end)

-- -----------------------------------------------
-- 23. coroutines (wrap/resume/yield — huge obf cost)
-- -----------------------------------------------
bench("coroutine wrap", function(i)
	local gen = coroutine.wrap(function()
		coroutine.yield(i * 2)
	end)
	local _ = gen()
end)

-- -----------------------------------------------
-- 24. tostring / tonumber churn
-- -----------------------------------------------
bench("tostring/tonumber", function(i)
	local s = tostring(i)
	local n = tonumber(s)
	local _ = n + 1
end)

-- -----------------------------------------------
-- 25. recursion — fib (stack frames, upvalue chains)
-- -----------------------------------------------
local function fib(n)
	if n <= 1 then return n end
	return fib(n-1) + fib(n-2)
end
bench("recursion fib(12)", function(_)
	local __ = fib(12)
end)

-- -----------------------------------------------
-- 26. mutual recursion (two functions calling each other)
-- -----------------------------------------------
local isEven, isOdd
isEven = function(n) if n == 0 then return true end return isOdd(n-1) end
isOdd  = function(n) if n == 0 then return false end return isEven(n-1) end
bench("mutual recursion", function(i)
	local _ = isEven(i % 20)
end)

-- -----------------------------------------------
-- 27. rawget / rawset vs normal access
-- -----------------------------------------------
local _raw = setmetatable({}, { __index = function() return 0 end })
bench("rawget/rawset", function(i)
	rawset(_raw, "k", i)
	local _ = rawget(_raw, "k")
end)

-- -----------------------------------------------
-- 28. type() checks (common guard pattern)
-- -----------------------------------------------
bench("type() checks", function(i)
	local vals = { i, tostring(i), true, nil, {}, i * 0.5 }
	local count = 0
	for j = 1, #vals do
		if type(vals[j]) == "number" then count = count + 1 end
	end
	local _ = count
end)

-- -----------------------------------------------
-- 29. string.byte / string.char
-- -----------------------------------------------
bench("byte/char", function(i)
	local c = string.char(65 + i % 26)
	local _ = string.byte(c)
end)

-- -----------------------------------------------
-- 30. table.concat (common for builders)
-- -----------------------------------------------
bench("table.concat", function(i)
	local parts = { "a", tostring(i), "b", tostring(i*2), "c" }
	local _ = table.concat(parts, "-")
end)

-- -----------------------------------------------
-- report
-- -----------------------------------------------
local total = 0
for _, v in pairs(res) do total = total + v end

-- score: based on total time across all 30 tests
-- baseline was tuned on a clean unobfuscated run
-- lower total = faster = higher score
-- 1000 = perfect (theoretical), 0 = catastrophically slow
-- sweet spot for unobfuscated code lands around 700-850
-- obfuscated code typically drops 100-400 points depending on tool
local BASELINE_GOOD  = 0.8   -- seconds, roughly "clean fast machine"
local BASELINE_BAD   = 6.0   -- seconds, roughly "heavily obfuscated dog"
local clamped = math.max(BASELINE_GOOD, math.min(BASELINE_BAD, total))
local score = math.floor(1000 * (1 - (clamped - BASELINE_GOOD) / (BASELINE_BAD - BASELINE_GOOD)))

local function grade(s)
	if s >= 900 then return "S  (clean / no overhead)"
	elseif s >= 750 then return "A  (light overhead)"
	elseif s >= 600 then return "B  (moderate overhead)"
	elseif s >= 400 then return "C  (heavy overhead)"
	elseif s >= 200 then return "D  (very heavy overhead)"
	else return "F  (something is very wrong)"
	end
end

print("")
print(rpad("TEST", 28) .. "TIME")
print(string.rep("-", 42))
for _, name in ipairs(ord) do
	print(rpad(name, 28) .. fmt(res[name]))
end
print(string.rep("-", 42))
print(rpad("TOTAL", 28) .. fmt(total))
print(string.rep("-", 42))
print(rpad("SCORE", 28) .. tostring(score) .. " / 1000")
print(rpad("GRADE", 28) .. grade(score))
print("")