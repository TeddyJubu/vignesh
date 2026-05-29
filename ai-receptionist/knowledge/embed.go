package knowledge

import _ "embed"

// SoulMD is Julia's persona (SOUL.md). Edit the markdown file and redeploy; migrate v7+ syncs to agent_notes.identity_soul.
//
//go:embed SOUL.md
var SoulMD string

// KnowledgeMD is the Epicware product/support knowledge base. Migrate v8+ syncs to agent_notes.client_instructions.
//
//go:embed KNOWLEDGE.md
var KnowledgeMD string
