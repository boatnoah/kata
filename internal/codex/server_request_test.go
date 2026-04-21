package codex

import (
	"encoding/json"
	"testing"

	"github.com/boatnoah/kata/internal/agent"
)

func TestIsServerInitiatedJSONRPCRequest(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		msg  rpcMessage
		want bool
	}{
		{
			name: "command execution approval",
			msg: rpcMessage{
				ID:     json.RawMessage(`42`),
				Method: methodItemCommandExecutionRequestApproval,
				Params: json.RawMessage(`{"threadId":"t","turnId":"u","itemId":"i"}`),
			},
			want: true,
		},
		{
			name: "response to client call",
			msg: rpcMessage{
				ID:     json.RawMessage(`1`),
				Result: json.RawMessage(`{"ok":true}`),
			},
			want: false,
		},
		{
			name: "error response",
			msg: rpcMessage{
				ID:    json.RawMessage(`2`),
				Error: &rpcError{Code: 1, Message: "x"},
			},
			want: false,
		},
		{
			name: "void result null",
			msg: rpcMessage{
				ID:     json.RawMessage(`3`),
				Result: json.RawMessage(`null`),
			},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isServerInitiatedJSONRPCRequest(tc.msg); got != tc.want {
				t.Fatalf("isServerInitiatedJSONRPCRequest() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCommandExecutionApprovalEvent(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"threadId": "th1",
		"turnId": "tn1",
		"itemId": "it1",
		"command": "ls -la",
		"cwd": "/tmp",
		"reason": "sandbox"
	}`)
	id := json.RawMessage(`99`)
	ev, err := commandExecutionApprovalEvent(id, raw)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != agent.EventApprovalRequired {
		t.Fatalf("Type = %v", ev.Type)
	}
	if ev.ThreadID != "th1" || ev.TurnID != "tn1" || ev.ItemID != "it1" {
		t.Fatalf("ids: %+v", ev)
	}
	if ev.Text != "ls -la" {
		t.Fatalf("Text = %q", ev.Text)
	}
	if string(ev.RPCID) != "99" {
		t.Fatalf("RPCID = %q", ev.RPCID)
	}
	if ev.Payload[approvalPayloadMethod] != methodItemCommandExecutionRequestApproval {
		t.Fatalf("method: %v", ev.Payload[approvalPayloadMethod])
	}
	if ev.Payload[approvalPayloadKind] != ApprovalKindCommandExecution {
		t.Fatalf("kind: %v", ev.Payload[approvalPayloadKind])
	}
	if ev.Payload["cwd"] != "/tmp" {
		t.Fatalf("cwd: %v", ev.Payload["cwd"])
	}
}

func TestSummarizeCommandExecutionApprovalFallback(t *testing.T) {
	t.Parallel()
	p := commandExecutionRequestApprovalParams{
		ThreadID: "t",
		TurnID:   "u",
		ItemID:   "i",
	}
	if got := summarizeCommandExecutionApproval(&p); got != "command execution approval" {
		t.Fatalf("got %q", got)
	}
	reason := "needs network"
	p.Reason = &reason
	if got := summarizeCommandExecutionApproval(&p); got != "needs network" {
		t.Fatalf("got %q", got)
	}
}

func TestFileChangeApprovalEvent(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"threadId":"a","turnId":"b","itemId":"c","reason":"extra writes"}`)
	ev, err := fileChangeApprovalEvent(json.RawMessage(`1`), raw)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Payload[approvalPayloadKind] != ApprovalKindFileChange {
		t.Fatalf("kind %v", ev.Payload[approvalPayloadKind])
	}
	if ev.Text != "extra writes" {
		t.Fatalf("text %q", ev.Text)
	}
}

func TestPermissionsApprovalEvent(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"threadId":"t","turnId":"u","itemId":"i","cwd":"/w","permissions":{"network":{"enabled":true}}}`)
	ev, err := permissionsApprovalEvent(json.RawMessage(`2`), raw)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Payload[approvalPayloadKind] != ApprovalKindPermissions {
		t.Fatalf("kind %v", ev.Payload[approvalPayloadKind])
	}
}

func TestDynamicToolCallEvent(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"threadId":"t","turnId":"u","callId":"c","tool":"mytool","arguments":{"x":1}}`)
	ev, err := dynamicToolCallEvent(json.RawMessage(`3`), raw)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Payload[approvalPayloadKind] != ApprovalKindDynamicToolCall {
		t.Fatalf("kind %v", ev.Payload[approvalPayloadKind])
	}
	if ev.Text != "mytool" {
		t.Fatalf("text %q", ev.Text)
	}
}

func TestMcpElicitationEvent(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"threadId":"t","turnId":null,"serverName":"srv","mode":"form","message":"Please confirm"}`)
	ev, err := mcpElicitationEvent(json.RawMessage(`4`), raw)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Payload[approvalPayloadKind] != ApprovalKindMcpElicitation {
		t.Fatalf("kind %v", ev.Payload[approvalPayloadKind])
	}
	if ev.TurnID != "" {
		t.Fatalf("turnId %q", ev.TurnID)
	}
}

func TestApplyPatchApprovalEvent(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"callId":"c1","conversationId":"th","fileChanges":{},"reason":"patch"}`)
	ev, err := applyPatchApprovalEvent(json.RawMessage(`5`), raw)
	if err != nil {
		t.Fatal(err)
	}
	if ev.ThreadID != "th" || ev.ItemID != "c1" {
		t.Fatalf("thread/item %+v", ev)
	}
	if ev.Payload[approvalPayloadKind] != ApprovalKindApplyPatch {
		t.Fatalf("kind %v", ev.Payload[approvalPayloadKind])
	}
}

func TestExecCommandApprovalEvent(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"callId":"c","command":["/bin/ls"],"conversationId":"th","cwd":"/","parsedCmd":[]}`)
	ev, err := execCommandApprovalEvent(json.RawMessage(`6`), raw)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Text != "/bin/ls" {
		t.Fatalf("text %q", ev.Text)
	}
	if ev.Payload[approvalPayloadKind] != ApprovalKindExecCommand {
		t.Fatalf("kind %v", ev.Payload[approvalPayloadKind])
	}
}

func TestToolRequestUserInputEvent(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"threadId":"t","turnId":"u","itemId":"i","questions":[{"id":"q1","header":"h","question":"Pick one?"}]}`)
	ev, err := toolRequestUserInputEvent(json.RawMessage(`7`), raw)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Text != "Pick one?" {
		t.Fatalf("text %q", ev.Text)
	}
}
