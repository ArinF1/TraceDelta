package main

func exampleScenario() []spanSpec {
	return []spanSpec{
		{
			Name:       "checkout.handle",
			SpanID:     rootSpanID,
			StartAfter: 0,
			Duration:   100_000_000,
			Kind:       2,
			StatusCode: 1,
		},
		{
			Name:       "payment.charge",
			SpanID:     "2222222222222222",
			ParentID:   rootSpanID,
			StartAfter: 10_000_000,
			Duration:   40_000_000,
			Kind:       3,
			StatusCode: 1,
		},
		{
			Name:       "cache.get",
			SpanID:     "3333333333333333",
			ParentID:   rootSpanID,
			StartAfter: 55_000_000,
			Duration:   10_000_000,
			Kind:       3,
			StatusCode: 1,
		},
	}
}
