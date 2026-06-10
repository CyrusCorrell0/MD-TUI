# MolTUI — project guide for Claude

Terminal molecular-dynamics viewer (think a tiny ChimeraX/VMD in the terminal).
Loads multi-frame PDB trajectories and renders them as half-block "pixel" art with
true-colour ANSI, with playback, rotation, zoom, cartoon/secondary-structure mode,
and GIF recording.

## Layout / module

- Self-contained Go module rooted at the repo root: `module github.com/CyrusCorrell0/MD-TUI`, go 1.22.
- Packages live at the repo root: `main.go`, `model/`, `pdb/`, `render/`, `ui/`.
- Run / build from the repo root:
  ```powershell
  go run .
  go build -o moltui.exe .
  ```
- The app scans the current working directory for `*.pdb` files. A sample
  (`128_path_000.pdb`) ships in the repo.

## Package map

- `main.go` — entry point. Wraps stdout in a synchronized-output writer (see Flicker) and
  starts the Bubble Tea program with the alt screen.
- `model/` — Bubble Tea models (Elm architecture: `Init`/`Update`/`View`).
  - `app.go` — top-level model, screen switching (file picker / viewer / error), async PDB load.
  - `filepicker.go` — `bubbles/list` of `.pdb` files in the cwd.
  - `viewer.go` — the 3D viewer: camera state, keybinds, frame playback, GIF recording trigger.
- `render/` — pure rendering, no Bubble Tea deps.
  - `pixel.go` — `PixelBuffer` (half-block 2×-vertical-resolution canvas) + projection + ANSI serialise.
  - `buffer.go` — older single-resolution `ScreenBuffer` (kept for reference; viewer uses PixelBuffer).
  - `cartoon.go` — secondary-structure cartoon ribbon render.
  - `record.go` — render frames to images and encode an animated GIF (stdlib `image/gif` only).
  - `modes.go` / `colors.go` / `project.go` — render modes, CPK colours/VdW radii, 3D math.
- `pdb/` — PDB parser (`parser.go`), secondary-structure parsing (`secondary.go`), types (`types.go`).
- `ui/` — lipgloss styles (`styles.go`) and key bindings (`keybindings.go`).

## Rendering model (important)

- The canvas is a `render.PixelBuffer`: each terminal cell holds TWO vertical pixels using
  `▀`/`▄` half-blocks — top pixel = foreground colour, bottom pixel = background colour.
  So pixel height = terminal rows × 2. Y projection accounts for this (factor 1.0, not 0.5).
- `PixelBuffer.Render()` emits true-colour (`\x1b[38;2;…m` / `\x1b[48;2;…m`) and run-length
  compresses runs of identical fg/bg pairs to keep output small.
- Depth buffering: larger `Depth` wins (painter's algorithm via z-test in `SetPixel`).

## Conventions / gotchas

- The background is BLACK everywhere. Black is defined ONCE per layer:
  - lipgloss styles: `ui.ColorBackground` in `ui/styles.go`.
  - pixel canvas fill: `pixelBg{R,G,B}` constants in `render/pixel.go`.
  - GIF background: passed into `render/record.go`.
  If you change "black", change all three or they will visibly disagree.
- Viewer keys are matched on `msg.String()` directly in `viewer.go` (the `ui.ViewerKeyMap`
  exists but the viewer does not route through it). Add new keys to the switch in `viewer.Update`
  AND to the status-bar help string AND to the README/this file.
- Only re-render when `m.dirty` is set, then cache the string in `m.cachedView`. Set `dirty=true`
  on any state change that affects the image. Returning an unchanged View string lets the
  renderer skip the frame (no flush) — this is load-bearing for not flickering when idle.
- No emojis anywhere (global user rule).

## Flicker (do not regress)

Bubble Tea v0.26.6's standard renderer erases+repaints changed lines WITHOUT wrapping a frame
in terminal synchronized-output markers, so during full-canvas repaints (playback / rotation)
the terminal can present intermediate blanked-line states → flicker, especially on Windows Terminal.

Fix lives in `main.go`: stdout is wrapped in a `syncWriter` that brackets every renderer flush
(one `Write` per frame) with DEC private mode 2026 — `\x1b[?2026h` (begin) … `\x1b[?2026l` (end).
The wrapper embeds `*os.File` so it still satisfies bubbletea's `term.File` check (Fd()), which is
required for Windows VT processing and terminal sizing. Do NOT replace it with a plain io.Writer —
that disables VT processing on Windows and the escapes print literally.

If you upgrade Bubble Tea to a version with built-in synchronized output, the wrapper becomes
redundant and can be removed.

## Recording

`R` in the viewer exports the WHOLE trajectory as an animated GIF next to the source PDB
(`<name>_recording_<timestamp>.gif`), using the current camera (rotation/zoom/pan/mode/cartoon)
and the current playback FPS for the frame delay. Encoding runs in a `tea.Cmd` (off the UI loop)
and reports the path (or error) in the status bar. Uses only the Go standard library
(`image`, `image/gif`, `image/color/palette`, `image/draw`) — no new dependencies.

## Controls

File picker: `up/down` or `j/k` navigate, `enter` open, `q`/`esc` quit, `ctrl+c` quit.
Viewer: `arrows`/`hjkl`/`wasd` rotate, `HJKL` pan, `+/-` zoom, `space` play/pause,
`./,` next/prev frame, `[`/`]` slower/faster, `m` cycle mode, `c` cartoon, `e` toggle H,
`tab` sidebar, `R` record GIF, `0` reset view, `q`/`esc` back, `ctrl+c` quit.
