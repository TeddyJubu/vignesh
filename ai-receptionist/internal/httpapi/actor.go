package httpapi

import "context"

type Actor struct {
	Phone       string          `json:"phone"`
	Role        string          `json:"role"`
	Permissions map[string]bool `json:"permissions,omitempty"`
	Source      string          `json:"-"`
}

type actorKey struct{}

func ContextWithActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, a)
}

func ActorFromContext(ctx context.Context) (Actor, bool) {
	v := ctx.Value(actorKey{})
	if v == nil {
		return Actor{}, false
	}
	a, ok := v.(Actor)
	return a, ok
}

