package domain

import "testing"

func TestCanTransitionTask(t *testing.T) {
	tests := []struct {
		from string
		to   string
		want bool
	}{
		{TaskStatusBacklog, TaskStatusReady, true},
		{TaskStatusReady, TaskStatusInProgress, true},
		{TaskStatusInProgress, TaskStatusReview, true},
		{TaskStatusReview, TaskStatusDone, true},
		{TaskStatusReview, TaskStatusInProgress, true},
		{TaskStatusBlocked, TaskStatusReady, true},
		{TaskStatusDone, TaskStatusInProgress, false},
		{TaskStatusReady, TaskStatusDone, false},
	}
	for _, test := range tests {
		if got := CanTransitionTask(test.from, test.to); got != test.want {
			t.Fatalf("CanTransitionTask(%s, %s) = %v, want %v", test.from, test.to, got, test.want)
		}
	}
}
