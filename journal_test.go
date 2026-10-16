package saga

import (
	"context"
	"errors"
	"testing"
)

func TestExecutionJournal(t *testing.T) {
	first := Do("reserve", func(
		context.Context, struct{},
	) (any, error) {
		return nil, nil
	})

	failure := errors.New("payment failed")

	second := Do("charge", func(
		context.Context, struct{},
	) (any, error) {
		return nil, failure
	})

	root := Sequence(first, second)

	journal, err := execute(context.Background(), root, struct{}{})

	if !errors.Is(err, failure) {
		t.Fatalf("expected payment error, got %v", err)
	}

	if journal.Succeeded {
		t.Fatal("sequence should not be marked successful")
	}

	if len(journal.Children) != 2 {
		t.Fatalf("expected 2 journal records, got %d", len(journal.Children))
	}

	if !journal.Children[0].Succeeded {
		t.Fatal("first step should be marked successful")
	}

	if journal.Children[1].Succeeded {
		t.Fatal("failed step should not be marked successful")
	}
}
