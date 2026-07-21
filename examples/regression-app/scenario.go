package main

func exampleScenario() []spanSpec {
	return []spanSpec{
		{
			Name:       "checkout.handle",
			SpanID:     rootSpanID,
			StartAfter: 0,
			Duration:   100_000_000,
			Kind:       2,
			StatusCode: 2,
			ErrorType:  "checkout.declined",
		},
		{
			Name:       "payment.charge",
			SpanID:     "2222222222222222",
			ParentID:   rootSpanID,
			StartAfter: 10_000_000,
			Duration:   70_000_000,
			Kind:       3,
			StatusCode: 1,
		},
		{
			Name:       "inventory.reserve",
			SpanID:     "4444444444444444",
			ParentID:   rootSpanID,
			StartAfter: 60_000_000,
			Duration:   15_000_000,
			Kind:       3,
			StatusCode: 1,
		},
	}
}
