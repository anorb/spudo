package messagereaction

import "github.com/anorb/spudo"

func init() {
	spudo.AddMessageReactionPlugin("reacts to ok", []string{"ok"}, []string{"👌"})
}
