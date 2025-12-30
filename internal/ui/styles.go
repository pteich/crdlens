package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	PrimaryColor   = lipgloss.Color("#7D56F4")
	SecondaryColor = lipgloss.Color("#F780E2")
	AccentColor    = lipgloss.Color("#00D9FF")
	White          = lipgloss.Color("#FFFFFF")
	Gray           = lipgloss.Color("#767676")
	Black          = lipgloss.Color("#000000")

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(White).
			Background(PrimaryColor).
			Padding(0, 1)

	StatusStyle = lipgloss.NewStyle().
			Foreground(White).
			Background(Gray).
			Padding(0, 1)

	AppStyle = lipgloss.NewStyle().Padding(1, 2)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			MarginBottom(1)

	SpinnerStyle = lipgloss.NewStyle().Foreground(SecondaryColor)

	StatusBarMainStyle = lipgloss.NewStyle().
				Foreground(White).
				Background(lipgloss.Color("#3C3C3C")).
				Padding(0, 1)

	StatusBarExtraStyle = lipgloss.NewStyle().
				Foreground(White).
				Background(lipgloss.Color("#575757")).
				Padding(0, 1)
)
