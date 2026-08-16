package pantheon

import "testing"

func TestChunkNexusConversationSeparatesUserAndAgentBlocks(t *testing.T) {
	input := ">[!nexus_user] **User**\n> My personal contribution is here.\n>\n>[!nexus_agent] **Assistant**\n> Contextual reply here.\n"
	chunks := chunkNexusConversation("Nexus/example.md", input)
	if len(chunks) != 2 {
		t.Fatalf("chunks=%d", len(chunks))
	}
	if chunks[0].Authorship != "nexus_user" || chunks[0].Text != "My personal contribution is here." {
		t.Fatalf("user=%#v", chunks[0])
	}
	if chunks[1].Authorship != "nexus_agent" || chunks[1].Text != "Contextual reply here." {
		t.Fatalf("agent=%#v", chunks[1])
	}
}
