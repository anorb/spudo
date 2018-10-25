package messagereaction

import "github.com/anorb/spudo"

func init() {
	spudo.AddMessageReactionPlugins("reacts to ok", []string{"ok"}, []string{"👌"})
}
