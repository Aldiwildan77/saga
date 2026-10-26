package saga

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRunNilRoot(t *testing.T) {
	err := Run[struct{}](context.Background(), nil, struct{}{})

	if !errors.Is(err, ErrorRootEmpty) {
		t.Fatalf("expected root empty error, got %v", err)
	}
}

func TestTaskWithoutExecuteFunc(t *testing.T) {
	err := Run(context.Background(), Do[struct{}, struct{}]("missing", nil), struct{}{})

	if err == nil || !strings.Contains(err.Error(), `step "missing" has no execute function`) {
		t.Fatalf("expected missing execute error, got %v", err)
	}
}

func TestTaskInputTypeMismatch(t *testing.T) {
	flow := Do("typed", func(context.Context, int) (int, error) {
		t.Fatal("step should not execute with mismatched input")
		return 0, nil
	})

	err := Run(context.Background(), flow, "not an int")

	if err == nil || !strings.Contains(err.Error(), "input type mismatch") {
		t.Fatalf("expected input type mismatch, got %v", err)
	}
}

func TestSequenceInputTypeMismatchWrapsSequenceError(t *testing.T) {
	flow := Sequence(
		Do("produce", func(context.Context, struct{}) (string, error) {
			return "text", nil
		}),
		Do("consume", func(context.Context, int) (int, error) {
			t.Fatal("consume should not execute with mismatched input")
			return 0, nil
		}),
	)

	err := Run(context.Background(), flow, struct{}{})

	if !errors.Is(err, ErrorSequenceType) {
		t.Fatalf("expected sequence error, got %v", err)
	}

	if !strings.Contains(err.Error(), "input type mismatch") {
		t.Fatalf("expected input type mismatch detail, got %v", err)
	}
}

func TestTaskErrorWrapsStepName(t *testing.T) {
	failure := errors.New("card declined")

	err := Run(context.Background(), Do("charge", func(
		context.Context, struct{},
	) (struct{}, error) {
		return struct{}{}, failure
	}), struct{}{})

	if !errors.Is(err, failure) {
		t.Fatalf("expected original error, got %v", err)
	}

	want := `step "charge" failed: card declined`
	if err.Error() != want {
		t.Fatalf("got %q, want %q", err.Error(), want)
	}
}

func TestParallelFailureJoinsAllBranches(t *testing.T) {
	first := errors.New("inventory down")
	second := errors.New("payment down")

	flow := Parallel(
		Do("inventory", func(context.Context, struct{}) (struct{}, error) {
			return struct{}{}, first
		}),
		Do("payment", func(context.Context, struct{}) (struct{}, error) {
			return struct{}{}, second
		}),
	)

	err := Run(context.Background(), flow, struct{}{})

	if !errors.Is(err, ErrorParallelType) {
		t.Fatalf("expected parallel error, got %v", err)
	}

	if !errors.Is(err, first) || !errors.Is(err, second) {
		t.Fatalf("expected both branch errors, got %v", err)
	}

	if !strings.Contains(err.Error(), "branch 0") || !strings.Contains(err.Error(), "branch 1") {
		t.Fatalf("expected branch indexes in error, got %v", err)
	}
}

func TestUnsupportedNodeKind(t *testing.T) {
	node := &Node[struct{}, struct{}]{name: "bogus", kind: Kind(99)}

	err := Run(context.Background(), node, struct{}{})

	if !errors.Is(err, ErrorNoneKind) {
		t.Fatalf("expected unsupported kind error, got %v", err)
	}
}

func TestUnsupportedCompensationMode(t *testing.T) {
	var compensated atomic.Int32

	task := Do("reserve", func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}).Compensate(CompensationPlan[struct{}]{
		mode: Mode(99),
		actions: []Action[struct{}, any]{
			func(context.Context, struct{}) (any, error) {
				compensated.Add(1)
				return nil, nil
			},
		},
	})

	err := Run(context.Background(), Sequence(
		task,
		Do("charge", func(context.Context, struct{}) (struct{}, error) {
			return struct{}{}, errors.New("declined")
		}),
	), struct{}{})

	if !errors.Is(err, ErrorNoneMode) {
		t.Fatalf("expected unsupported mode error, got %v", err)
	}

	if compensated.Load() != 0 {
		t.Fatal("compensation should not run with unsupported mode")
	}
}

func TestCompensationActionErrorIsWrapped(t *testing.T) {
	releaseErr := errors.New("warehouse offline")

	task := Do("reserve", func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}).Compensate(SequentialCompensations(
		func(context.Context, struct{}) (any, error) {
			return nil, releaseErr
		},
	))

	err := Run(context.Background(), Sequence(
		task,
		Do("charge", func(context.Context, struct{}) (struct{}, error) {
			return struct{}{}, errors.New("declined")
		}),
	), struct{}{})

	if !errors.Is(err, ErrorCompensation) {
		t.Fatalf("expected compensation sentinel, got %v", err)
	}

	if !errors.Is(err, releaseErr) {
		t.Fatalf("expected original compensation error, got %v", err)
	}
}

func TestSequentialCompensationJoinsAllErrors(t *testing.T) {
	var ran []string
	errA := errors.New("a failed")
	errB := errors.New("b failed")

	task := Do("reserve", func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}).Compensate(SequentialCompensations(
		func(context.Context, struct{}) (any, error) {
			ran = append(ran, "a")
			return nil, errA
		},
		func(context.Context, struct{}) (any, error) {
			ran = append(ran, "b")
			return nil, errB
		},
	))

	err := Run(context.Background(), Sequence(
		task,
		Do("charge", func(context.Context, struct{}) (struct{}, error) {
			return struct{}{}, errors.New("declined")
		}),
	), struct{}{})

	if !errors.Is(err, errA) || !errors.Is(err, errB) {
		t.Fatalf("expected both compensation errors, got %v", err)
	}

	// Sequential compensations run in reverse order and keep going after errors.
	if strings.Join(ran, ",") != "b,a" {
		t.Fatalf("got order %v, want [b a]", ran)
	}
}

func TestParallelCompensationJoinsAllErrors(t *testing.T) {
	errA := errors.New("a failed")
	errB := errors.New("b failed")

	task := Do("reserve", func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}).Compensate(ParallelCompensations(
		func(context.Context, struct{}) (any, error) {
			return nil, errA
		},
		func(context.Context, struct{}) (any, error) {
			return nil, errB
		},
	))

	err := Run(context.Background(), Sequence(
		task,
		Do("charge", func(context.Context, struct{}) (struct{}, error) {
			return struct{}{}, errors.New("declined")
		}),
	), struct{}{})

	if !errors.Is(err, errA) || !errors.Is(err, errB) {
		t.Fatalf("expected both compensation errors, got %v", err)
	}
}

func TestFailedTaskIsNotCompensated(t *testing.T) {
	var compensated atomic.Int32

	task := Do("charge", func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, errors.New("declined")
	}).Compensate(SequentialCompensations(
		func(context.Context, struct{}) (any, error) {
			compensated.Add(1)
			return nil, nil
		},
	))

	err := Run(context.Background(), task, struct{}{})

	if err == nil {
		t.Fatal("expected failure")
	}

	if compensated.Load() != 0 {
		t.Fatal("failed task should not be compensated")
	}
}

func TestCompensateNodeInputTypeMismatch(t *testing.T) {
	task := Do("reserve", func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}).Compensate(SequentialCompensations(
		func(context.Context, struct{}) (any, error) {
			return nil, nil
		},
	))

	err := task.compensateNode(context.Background(), "wrong type")

	if err == nil || !strings.Contains(err.Error(), "input type mismatch") {
		t.Fatalf("expected input type mismatch, got %v", err)
	}
}

func TestCompensateNodeWithoutActions(t *testing.T) {
	task := Do("reserve", func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	})

	if err := task.compensateNode(context.Background(), "wrong type"); err != nil {
		t.Fatalf("node without compensation should be a no-op, got %v", err)
	}
}

func TestCompensationRunsAfterContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var compensationCtxErr error

	flow := Sequence(
		Do("reserve", func(context.Context, struct{}) (struct{}, error) {
			return struct{}{}, nil
		}).Compensate(SequentialCompensations(
			func(c context.Context, _ struct{}) (any, error) {
				compensationCtxErr = c.Err()
				return nil, nil
			},
		)),
		Do("charge", func(context.Context, struct{}) (struct{}, error) {
			cancel()
			return struct{}{}, errors.New("declined")
		}),
	)

	err := Run(ctx, flow, struct{}{})

	if err == nil {
		t.Fatal("expected failure")
	}

	if compensationCtxErr != nil {
		t.Fatalf("compensation should run on a non-cancelled context, got %v", compensationCtxErr)
	}
}

func TestCancellationBetweenStepsStopsSequence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var secondRan atomic.Bool

	flow := Sequence(
		Do("first", func(context.Context, struct{}) (struct{}, error) {
			cancel()
			return struct{}{}, nil
		}),
		Do("second", func(context.Context, struct{}) (struct{}, error) {
			secondRan.Store(true)
			return struct{}{}, nil
		}),
	)

	err := Run(ctx, flow, struct{}{})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}

	if secondRan.Load() {
		t.Fatal("second step should not run after cancellation")
	}
}

func TestCancelledBeforeParallelSkipsBranches(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	flow := Parallel(
		Do("a", func(context.Context, struct{}) (struct{}, error) {
			t.Fatal("branch should not run on cancelled context")
			return struct{}{}, nil
		}),
		Do("b", func(context.Context, struct{}) (struct{}, error) {
			t.Fatal("branch should not run on cancelled context")
			return struct{}{}, nil
		}),
	)

	err := Run(ctx, flow, struct{}{})

	// The group checks ctx before fanning out, so no branch error is produced.
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestRollbackNilRecord(t *testing.T) {
	if err := rollback(context.Background(), nil); err != nil {
		t.Fatalf("nil record should roll back to nil, got %v", err)
	}
}

func TestRecordHasNode(t *testing.T) {
	var nilRecord *Record

	if nilRecord.HasNode() {
		t.Fatal("nil record should not have a node")
	}

	if (&Record{}).HasNode() {
		t.Fatal("record without node should not have a node")
	}

	if !(&Record{Node: Do("x", func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	})}).HasNode() {
		t.Fatal("record with node should have a node")
	}
}
