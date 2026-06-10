# Plan: Terminal PDB Molecular Viewer (Bubble Tea)
        2
        3 ## Context
        4
        5 Build a Go TUI using Charmbracelet's Bubble Tea that renders PDB (Protein Data Bank) molecular structure files in the terminal — a VMD/NGLview-style viewer that w
          orks entirely in ASCII/ANSI. The project directory `C:\Users\iamcy\Desktop\MD\TUI` is currently empty.
        6
        7 ---
        8
        9 ## Project Structure
       10
       11 ```
       12 C:\Users\iamcy\Desktop\MD\TUI\
       13 ├── go.mod
       14 ├── main.go
       15 ├── pdb/
       16 │   ├── types.go      -- Atom, Frame, Bond, Molecule structs
       17 │   └── parser.go     -- PDB file parsing (ATOM/HETATM/MODEL/ENDMDL/CONECT)
       18 ├── render/
       19 │   ├── buffer.go     -- ScreenBuffer with Z-buffer + Render() hot path
       20 │   ├── project.go    -- 3D math: Vec3, rotation matrices, orthographic projection
       21 │   ├── modes.go      -- Space-fill, ball-and-stick, wireframe renderers
       22 │   └── colors.go     -- CPK color table, VDW radii, shading ramp
       23 ├── ui/
       24 │   ├── styles.go     -- All lipgloss styles (dark VMD-like theme)
       25 │   └── keybindings.go -- All key.Binding definitions (viewer + file picker)
       26 └── model/
       27     ├── app.go        -- Root AppModel, screen routing, WindowSizeMsg
       28     ├── filepicker.go -- bubbles/list wrapper for .pdb file selection
       29     └── viewer.go     -- 3D viewer state machine, dirty-flag render cache
       30 ```
       31
       32 ## Dependencies (go.mod)
       33
       34 ```
       35 module github.com/user/moltui
       36 go 1.22
       37 require (
       38     github.com/charmbracelet/bubbletea v0.26.6
       39     github.com/charmbracelet/bubbles v0.18.0
       40     github.com/charmbracelet/lipgloss v0.11.0
       41 )
       42 ```
       43
       44 ---
       45
       46 ## Key Types
       47
       48 ### pdb/types.go
       49 ```go
       50 type Atom struct {
       51     Serial  int; Name, ResName string; ChainID byte; ResSeq int
       52     X, Y, Z float64; Element string; IsHet bool
       53 }
       54 type Bond struct{ A, B int }     // atom serials
       55 type Frame struct{ Atoms []Atom; Bonds []Bond }
       56 type Molecule struct {
       57     Title  string; Frames []Frame
       58     MinX,MaxX, MinY,MaxY, MinZ,MaxZ float64
       59     CenterX,CenterY,CenterZ float64; Extent float64
       60 }
       61 ```
       62
       63 ### render/buffer.go
       64 ```go
       65 type Cell struct{ Ch rune; FG lipgloss.Color; Depth float64 }
       66 type ScreenBuffer struct{ Width,Height int; Cells [][]Cell }
       67 // Set() does z-test before writing. Render() joins rows with lipgloss per-cell styling.
       68 ```
       69
       70 ### model/viewer.go
       71 ```go
       72 type RenderMode int  // ModeSpaceFill / ModeBallStick / ModeWireframe
       73 type ViewerModel struct {
       74     width,height, canvasW,canvasH, sidebarW int
       75     mol *pdb.Molecule; buf *render.ScreenBuffer
       76     rotX,rotY, zoom, panX,panY float64
       77     frameIdx int; playing bool; playFPS int
       78     mode RenderMode; showH, showSidebar bool
       79     dirty bool; cachedView string
       80 }
       81 ```
       82
       83 ---
       84
       85 ## Screen Layout
       86
       87 ### File Picker (full screen)
       88 ```
       89 ┌─────────────────────────────────────────────┐
       90 │  Mol TUI  [cwd path]                        │  1 row header
       91 ├─────────────────────────────────────────────┤
       92 │  > protein.pdb          23.4 KB             │  height-5 rows list
       93 │    trajectory.pdb        1.2 MB             │
       94 ├─────────────────────────────────────────────┤
       95 │  ↑↓/jk navigate   enter select   q quit    │  2 row help
       96 └─────────────────────────────────────────────┘
       97 ```
       98
       99 ### Viewer (full screen)
      100 ```
      101 ┌─────────────────────────────────┬────────────┐
      102 │                                 │ MOLECULE   │  canvasH rows
      103 │      3D RENDER CANVAS           │ File: ...  │
      104 │   (canvasW = width-29)          │ Atoms: 123 │  sidebarW = 28
      105 │                                 │ Mode: CPK  │
      106 │                                 │ Frame: 1/n │
      107 │                                 │ Zoom: 2.3x │
      108 └─────────────────────────────────┴────────────┤
      109 │  [hjkl/arrows] rotate  [HJKL] pan  [+/-] zoom│  3 row status bar
      110 │  [space] play  [.,] frame  [m] mode  [q] back│
      111 └─────────────────────────────────────────────┘
      112 ```
      113
      114 ---
      115
      116 ## Render Pipeline
      117
      118 **Render modes (auto-selected by atom count, overridable with `m`):**
      119 - SpaceFill (< 5000 atoms): CPK spheres with Lambert shading, vdW radii
      120 - BallStick (5000–20000): smaller spheres + Bresenham bond lines
      121 - Wireframe (> 20000): C-alpha trace / bond skeleton only
      122
      123 **Per-frame render steps:**
      124 1. Clear ScreenBuffer (zero depth, space char)
      125 2. For each atom: apply `RotateYX(pos - center, rotY, rotX)`, then `Project()` → screen xy
      126 3. For SpaceFill: rasterize disk, compute sphere normal per pixel, Lambert = `dot(normal, lightDir)`, map to ShadingRamp char `" .,:-=+*#%@"`, Z-test then write
      127 4. For BallStick: same + Bresenham lines between bonded atoms with depth interpolation
      128 5. For Wireframe: Bresenham lines only
      129 6. `buf.Render()` → string → `m.cachedView`
      130
      131 **Projection (orthographic):**
      132 ```go
      133 sx = canvasW/2 + int((rotated.X)*zoom) + panX
      134 sy = canvasH/2 - int((rotated.Y)*zoom*0.5) + panY  // 0.5 for 2:1 cell aspect
      135 ```
      136
      137 **CPK colors (lipgloss hex):** C=#909090, N=#4444FF, O=#FF4444, S=#FFFF44, H=#FFFFFF, P=#FFA500
      138
      139 ---
      140
      141 ## Keybindings
      142
      143 | Key(s) | Action |
      144 |--------|--------|
      145 | `←→↑↓` / `hjkl` / `wasd` | Rotate model (5°/press) |
      146 | `HJKL` (shift) | Pan camera |
      147 | `+` / `-` | Zoom in/out (×1.15) |
      148 | `space` | Play/pause trajectory |
      149 | `.` / `,` | Next/previous frame |
      150 | `]` / `[` | Faster/slower playback FPS |
      151 | `m` | Cycle render mode (SpaceFill→BallStick→Wireframe) |
      152 | `e` | Toggle hydrogen visibility |
      153 | `tab` | Toggle sidebar |
      154 | `0` | Reset rotation/pan/zoom |
      155 | `q` / `esc` | Back to file picker |
      156 | `ctrl+c` | Quit |
      157
      158 ---
      159
      160 ## PDB Parser Details
      161
      162 Column positions (0-indexed, always bounds-check `len(line)`):
      163 - Record type: `[0:6]`
      164 - Serial: `[6:11]`
      165 - Atom name: `[12:16]` → TrimSpace
      166 - Residue name: `[17:20]`
      167 - Chain ID: `[21]`
      168 - X,Y,Z: `[30:38]`, `[38:46]`, `[46:54]`
      169 - Element: `[76:78]` (optional, fallback: infer from Name)
      170
      171 Multi-frame: accumulate atoms per MODEL block. If no MODEL records, whole file = single frame. Skip zero-atom frames. After all frames parsed, compute bounding bo
          x from frame[0] atoms.
      172
      173 Bond inference when CONECT absent: spatial grid bucketed at 2Å cells; bonded if distance < 2.1Å (skip H-H).
      174
      175 ---
      176
      177 ## State Machine
      178
      179 ```
      180 NewAppModel() → Init() scans cwd → ScreenFilePicker
      181     Enter on file → LoadMoleculeCmd (goroutine)
      182                   → MoleculeLoadedMsg → ScreenViewer
      183     q/esc in viewer → ScreenFilePicker
      184     ctrl+c anywhere → tea.Quit
      185 ```
      186
      187 Messages: `MoleculeLoadedMsg`, `MoleculeErrMsg`, `TickMsg`, `BackToPickerMsg`
      188
      189 `TickMsg` drives trajectory playback: chain `tea.Tick(1s/fps, ...)`, increment `frameIdx`, set `dirty=true`, reschedule only if `playing == true`.
      190
      191 Dirty flag: set on any state change (rotation, zoom, pan, frame change, resize, mode change). `View()` re-renders to `cachedView` when `dirty`, then returns cache
          d string. Since Bubble Tea models are value types, `dirty` and `cachedView` are both struct fields updated within `View()`.
      192
      193 ---
      194
      195 ## Styling (dark VMD theme)
      196
      197 ```go
      198 ColorBackground = "#0D1117"  // near-black
      199 ColorBorder     = "#30363D"  // subtle gray
      200 ColorAccent     = "#58A6FF"  // GitHub blue (canvas border glows this)
      201 ColorMuted      = "#8B949E"
      202 ColorHighlight  = "#F0F6FC"
      203 ```
      204
      205 Canvas gets a rounded blue border. Sidebar gets a subtle gray rounded border. Status bar is a dark strip at the bottom.
      206
      207 ---
      208
      209 ## Edge Cases
      210
      211 - Lines shorter than 54 chars: skip atom silently
      212 - No MODEL/ENDMDL: treat whole file as one frame
      213 - No PDB files in cwd: show empty-state message in file list
      214 - Canvas too small (< 10×5): show "Terminal too small" instead of rendering
      215 - `mol.Extent == 0`: default zoom = `canvasH / 3`
      216 - `TickMsg` after playback stopped: check `m.playing`, do not reschedule
      217
      218 ---
      219
      220 ## Implementation Order
      221
      222 1. `pdb/types.go` + `pdb/parser.go`
      223 2. `render/colors.go` + `render/project.go`
      224 3. `render/buffer.go`
      225 4. `render/modes.go` (Wireframe first, then SpaceFill, then BallStick)
      226 5. `ui/styles.go` + `ui/keybindings.go`
      227 6. `model/filepicker.go`
      228 7. `model/viewer.go` (Wireframe only first)
      229 8. `model/app.go`
      230 9. `main.go`
      231 10. Tune zoom defaults, colors, FPS
