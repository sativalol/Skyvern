package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	bolt "go.etcd.io/bbolt"
)

func (m *Model) dbReloadBuckets() {
	var db *bolt.DB
	if m.dbSelIdx == 0 {
		db = m.db.BoltDB()
	} else {
		db = m.mgr.PalantirDB()
	}

	m.dbBuckets = nil
	if db == nil {
		m.dbBuckets = []string{"(no database connection)"}
		return
	}

	_ = db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
			m.dbBuckets = append(m.dbBuckets, string(name))
			return nil
		})
	})
	sort.Strings(m.dbBuckets)
	if m.dbBktIdx >= len(m.dbBuckets) {
		m.dbBktIdx = 0
	}
	m.dbReloadKeys()
}

func (m *Model) dbReloadKeys() {
	var db *bolt.DB
	if m.dbSelIdx == 0 {
		db = m.db.BoltDB()
	} else {
		db = m.mgr.PalantirDB()
	}

	m.dbKeys = nil
	m.dbValue = ""
	if db == nil || len(m.dbBuckets) == 0 || m.dbBktIdx >= len(m.dbBuckets) {
		return
	}

	bucketName := m.dbBuckets[m.dbBktIdx]

	_ = db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(bucketName))
		if bkt == nil {
			return nil
		}

		filter := strings.ToLower(m.dbSearch.Value())

		return bkt.ForEach(func(k, _ []byte) error {
			keyStr := string(k)
			if filter == "" || strings.Contains(strings.ToLower(keyStr), filter) {
				m.dbKeys = append(m.dbKeys, keyStr)
			}
			return nil
		})
	})

	sort.Strings(m.dbKeys)

	if m.dbKeyIdx >= len(m.dbKeys) {
		m.dbKeyIdx = 0
	}
	m.dbReloadValue()
}

func (m *Model) dbReloadValue() {
	var db *bolt.DB
	if m.dbSelIdx == 0 {
		db = m.db.BoltDB()
	} else {
		db = m.mgr.PalantirDB()
	}

	m.dbValue = ""
	if db == nil || len(m.dbBuckets) == 0 || m.dbBktIdx >= len(m.dbBuckets) || len(m.dbKeys) == 0 || m.dbKeyIdx >= len(m.dbKeys) {
		return
	}

	bucketName := m.dbBuckets[m.dbBktIdx]
	keyName := m.dbKeys[m.dbKeyIdx]

	_ = db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(bucketName))
		if bkt == nil {
			return nil
		}
		val := bkt.Get([]byte(keyName))
		if val == nil {
			m.dbValue = "(nil)"
			return nil
		}

		// Try parsing as JSON for pretty printing
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, val, "", "  "); err == nil {
			m.dbValue = pretty.String()
		} else {
			// If not JSON, show as string if printable, otherwise format hex/bytes
			m.dbValue = string(val)
		}
		return nil
	})
}

func (m Model) updateDbBrowser(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.dbSearching {
			switch msg.String() {
			case "enter", "esc":
				m.dbSearching = false
				m.dbReloadKeys()
			default:
				m.dbSearch, cmd = m.dbSearch.Update(msg)
				m.dbReloadKeys()
			}
			return m, cmd
		}

		switch msg.String() {
		case "tab":
			m.tab = (m.tab + 1) % 7
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		case "d", "D":
			m.dbSelIdx = (m.dbSelIdx + 1) % 2
			m.dbBktIdx = 0
			m.dbKeyIdx = 0
			m.dbReloadBuckets()
		case "/":
			m.dbSearching = true
			m.dbSearch.Focus()
			m.dbSearch.SetValue("")
			return m, nil
		case "left", "h":
			if m.dbPane > 0 {
				m.dbPane--
			}
		case "right", "l":
			if m.dbPane < 1 {
				m.dbPane++
			}
		case "up", "k":
			if m.dbPane == 0 {
				if m.dbBktIdx > 0 {
					m.dbBktIdx--
					m.dbKeyIdx = 0
					m.dbReloadKeys()
				}
			} else {
				if m.dbKeyIdx > 0 {
					m.dbKeyIdx--
					m.dbReloadValue()
				}
			}
		case "down", "j":
			if m.dbPane == 0 {
				if m.dbBktIdx < len(m.dbBuckets)-1 {
					m.dbBktIdx++
					m.dbKeyIdx = 0
					m.dbReloadKeys()
				}
			} else {
				if m.dbKeyIdx < len(m.dbKeys)-1 {
					m.dbKeyIdx++
					m.dbReloadValue()
				}
			}
		}
	}
	return m, nil
}

func (m Model) renderDbBrowser(mainWidth, contentHeight int, th Theme) string {
	titleStyle := lipgloss.NewStyle().Foreground(th.Accent).Bold(true).Underline(true)
	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Border).Padding(1)
	boxFocusStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.BorderFocus).Padding(1)
	selStyle := lipgloss.NewStyle().Foreground(th.Accent).Background(th.BorderFocus).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(th.Subtle)

	innerW := calcMainInnerWidth(mainWidth)
	innerH := contentHeight - 4
	if innerH < 5 {
		innerH = 5
	}

	// Columns widths
	bktColWidth := 24
	keyColWidth := 30
	valColWidth := innerW - bktColWidth - keyColWidth - 6
	if valColWidth < 20 {
		valColWidth = 20
	}

	// Database Header Selector
	dbName := "BOTS.DB (Bolt)"
	if m.dbSelIdx == 1 {
		dbName = "PALANTIR.DB (Bolt)"
	}
	dbHeader := fmt.Sprintf(" DATABASE: [ %s ] (Press [D] to Toggle)", dbName)
	if m.dbSearching {
		dbHeader += fmt.Sprintf(" | Search: %s", m.dbSearch.View())
	} else {
		dbHeader += " | Search: [/] Filter"
	}

	// 1. Render Buckets list
	var bktLines []string
	bktLines = append(bktLines, titleStyle.Render("BUCKETS"))
	for i, bkt := range m.dbBuckets {
		display := bkt
		if len(display) > bktColWidth-2 {
			display = display[:bktColWidth-5] + "..."
		}
		if i == m.dbBktIdx {
			if m.dbPane == 0 {
				bktLines = append(bktLines, selStyle.Render(" > "+display))
			} else {
				bktLines = append(bktLines, " > "+display)
			}
		} else {
			bktLines = append(bktLines, "   "+display)
		}
	}

	// 2. Render Keys list
	var keyLines []string
	keyLines = append(keyLines, titleStyle.Render("KEYS"))
	if len(m.dbKeys) == 0 {
		keyLines = append(keyLines, dimStyle.Render("  (empty)"))
	} else {
		for i, key := range m.dbKeys {
			display := key
			if len(display) > keyColWidth-2 {
				display = display[:keyColWidth-5] + "..."
			}
			if i == m.dbKeyIdx {
				if m.dbPane == 1 {
					keyLines = append(keyLines, selStyle.Render(" > "+display))
				} else {
					keyLines = append(keyLines, " > "+display)
				}
			} else {
				keyLines = append(keyLines, "   "+display)
			}
		}
	}

	// 3. Render Value block
	var valLines []string
	valLines = append(valLines, titleStyle.Render("VALUE DETAIL"))
	if m.dbValue == "" {
		valLines = append(valLines, dimStyle.Render("  (no key selected)"))
	} else {
		// Limit value to rendering height
		rawValLines := strings.Split(m.dbValue, "\n")
		for idx, line := range rawValLines {
			if idx > innerH-3 {
				valLines = append(valLines, dimStyle.Render("... (truncated)"))
				break
			}
			if len(line) > valColWidth {
				line = line[:valColWidth-3] + "..."
			}
			valLines = append(valLines, line)
		}
	}

	// Make each column match height
	padLines := func(lines []string, height int) string {
		for len(lines) < height {
			lines = append(lines, "")
		}
		if len(lines) > height {
			lines = lines[:height]
		}
		return strings.Join(lines, "\n")
	}

	colBkt := padLines(bktLines, innerH-2)
	colKey := padLines(keyLines, innerH-2)
	colVal := padLines(valLines, innerH-2)

	bktStyle := boxStyle
	if m.dbPane == 0 {
		bktStyle = boxFocusStyle
	}
	keyStyle := boxStyle
	if m.dbPane == 1 {
		keyStyle = boxFocusStyle
	}

	renderBkt := bktStyle.Width(bktColWidth).Height(innerH - 2).Render(colBkt)
	renderKey := keyStyle.Width(keyColWidth).Height(innerH - 2).Render(colKey)
	renderVal := boxStyle.Width(valColWidth).Height(innerH - 2).Render(colVal)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, renderBkt, renderKey, renderVal)

	mainContent := lipgloss.JoinVertical(lipgloss.Left, dbHeader, columns)

	return boxStyle.Width(innerW).Height(innerH).Render(mainContent)
}
