#!/usr/bin/env python3
"""Apply the experimental bare-metal port to freshly generated vendored sources.

Run through make prepare. Only the ignored vendor tree is mutated. Keep the
upstream event loop unchanged; all replacements are terminal/platform plumbing.
"""
from pathlib import Path
v=Path(__file__).resolve().parent/'vendor'
modules=(v/'modules.txt').read_text()
assert '# charm.land/bubbletea/v2 v2.1.0' in modules, 'port requires Bubble Tea v2.1.0'
exclude={
'charm.land/bubbletea/v2':['tty_unix.go','signals_unix.go','termios_unix.go'],
'github.com/charmbracelet/ultraviolet':['terminal_unix.go','tty_unix.go','winch_unix.go','terminal_tabdly.go','terminal_bsdly.go','poll_linux.go'],
'github.com/charmbracelet/x/term':['term_unix.go','term_unix_other.go'],
'github.com/muesli/cancelreader':['cancelreader_linux.go'],
}
fallback={
'charm.land/bubbletea/v2':['termios_other.go'],
'github.com/charmbracelet/ultraviolet':['terminal_other.go','tty_other.go','winch_other.go','terminal_tabdly_other.go','terminal_bsdly_other.go','poll_default.go'],
'github.com/charmbracelet/x/term':['term_other.go'],
'github.com/muesli/cancelreader':['cancelreader_default.go'],
}
for table,op in [(exclude,'!baremetal &&'),(fallback,'baremetal ||')]:
 for folder,names in table.items():
  for name in names:
   p=v/folder/name
   lines=p.read_text().splitlines()
   assert lines[0].startswith('//go:build ')
   lines[0]='//go:build '+op+' ('+lines[0][11:]+')'
   lines=[l for l in lines if not l.startswith('// +build')]
   p.write_text('\n'.join(lines)+'\n')
(v/'charm.land/bubbletea/v2/tty_baremetal.go').write_text('''//go:build baremetal

package tea

func (*Program) initInput() error { return nil }
const suspendSupported = false
func suspendProcess() {}
func (*Program) listenForResize(done chan struct{}) { close(done) }
''')
p=v/'charm.land/bubbletea/v2/cursed_renderer.go'
p.write_text('//go:build !baremetal\n\n'+p.read_text())
(v/'charm.land/bubbletea/v2/cursed_renderer_baremetal.go').write_text('''//go:build baremetal

package tea

import (
 "io"
 uv "github.com/charmbracelet/ultraviolet"
)

type cursedRenderer struct { nilRenderer }
func newCursedRenderer(io.Writer, []string, int, int) *cursedRenderer { return &cursedRenderer{} }
func (*cursedRenderer) setLogger(uv.Logger) {}
func (*cursedRenderer) setNoInput(bool) {}
func (*cursedRenderer) setOptimizations(bool, bool, bool) {}
''')
# Terminal readers are unavailable on bare metal: input arrives through Send.
p=v/'charm.land/bubbletea/v2/tty.go'
s=p.read_text()
start=s.index('func (p *Program) initInputReader(cancel bool) error {')
end=s.index('// waitForReadLoop',start)
original=s[start:end]
s=s[:start]+s[end:]
p.write_text(s)
(v/'charm.land/bubbletea/v2/input_terminal.go').write_text('//go:build !baremetal\n\npackage tea\n\nimport ("fmt"; uv "github.com/charmbracelet/ultraviolet")\n\n'+original)
(v/'charm.land/bubbletea/v2/input_baremetal.go').write_text('''//go:build baremetal

package tea

import "errors"
func (*Program) initInputReader(bool) error { return errors.New("bubbletea: bare metal requires WithInput(nil); inject messages with Send") }
''')
