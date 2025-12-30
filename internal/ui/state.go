package ui

// ViewState represents the different views in the application
type ViewState int

const (
	CRDListView ViewState = iota
	CRListView
	CRDetailView
	CRDSpecView
	HelpView
)
