package tui

import "testing"

func TestApprovalAcceptResult(t *testing.T) {
	t.Parallel()
	if _, ok := approvalAcceptResult(approvalKindCommandExecution); !ok {
		t.Fatal("command execution accept")
	}
	if _, ok := approvalAcceptResult(approvalKindPermissions); ok {
		t.Fatal("permissions accept should need profile")
	}
}

func TestApprovalDeclineResult(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{
		approvalKindCommandExecution,
		approvalKindFileChange,
		approvalKindApplyPatch,
		approvalKindExecCommand,
		approvalKindPermissions,
		approvalKindMcpElicitation,
		approvalKindToolUserInput,
		approvalKindDynamicToolCall,
	} {
		if _, ok := approvalDeclineResult(kind); !ok {
			t.Fatalf("decline %q", kind)
		}
	}
}

func TestApprovalAcceptSessionResult(t *testing.T) {
	t.Parallel()
	if _, ok := approvalAcceptSessionResult(approvalKindCommandExecution); !ok {
		t.Fatal("exec session")
	}
	if _, ok := approvalAcceptSessionResult(approvalKindApplyPatch); !ok {
		t.Fatal("patch session")
	}
	if _, ok := approvalAcceptSessionResult(approvalKindPermissions); ok {
		t.Fatal("permissions session unsupported")
	}
}
