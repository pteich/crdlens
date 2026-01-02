# CRDLens

CRDLens is a terminal-based explorer for Kubernetes Custom Resource Definitions (CRDs) and Custom Resources (CRs). It helps developers and operators inspect CRD schemas and manage resources efficiently from the command line with a focus on speed.

## Features

- **CRD Discovery**: List all valid CRDs in your cluster with resource counts.
- **Hierarchical Schema Explorer**: Drill down into complex CRD schemas (OpenAPI v3) with a tree-based view.
- **Resource Management**: Browse Custom Resources for any CRD with fuzzy filtering.
- **Deep Inspection**: View resource details including YAML configuration, Events, and a structured Fields view for exploring deeply nested data.
- **Namespace Awareness**: Easily switch between namespaces or view resources across all namespaces.

## Installation

### Prerequisites
- Go 1.21 or higher
- A running Kubernetes cluster and `kubectl` configured

### Build from source
```bash
git clone https://github.com/pteich/crdlens.git
cd crdlens
go build -o crdlens ./cmd/crdlens
mv crdlens /usr/local/bin/
```

## Usage

Simply run `crdlens` in your terminal. It will use your current kubeconfig context.

```bash
crdlens
```

### Keybindings

| Key | Action |
| --- | --- |
| `Enter` | Select item / Drill down into field |
| `Esc` / `Backspace` | Go back / Navigate up hierarchy |
| `?` | Toggle Help |
| `/` | Filter / Search |
| `n` | Switch Namespace |
| `r` | Refresh list |
| `f` | Toggle Flat/Hierarchical view (in CRD Spec) |
| `Tab` | Switch Views (e.g., Table, YAML, Events) |
| `q` / `Ctrl+C` | Quit |

## Screenshots

