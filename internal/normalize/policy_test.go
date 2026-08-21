package normalize

import "testing"

func TestClassify(t *testing.T) {
	t.Parallel()
	p := V1()

	tests := []struct {
		name string
		key  string
		want Class
	}{
		{name: "unknown fest_ key fails closed", key: "fest_bogus", want: Unknown},
		{name: "future routing field is unknown until tabled", key: "fest_model", want: Unknown},
		{name: "status is state", key: "fest_status", want: Strip},
		{name: "updated is state", key: "fest_updated", want: Strip},
		{name: "working dir is environment", key: "fest_working_dir", want: Strip},
		{name: "identity retained", key: "fest_id", want: Retain},
		{name: "plan shape retained", key: "fest_dependencies", want: Retain},
		{name: "constraint retained", key: "fest_gate_type", want: Retain},
		{name: "typing retained", key: "fest_phase_type", want: Retain},
		{name: "routing intent retained", key: "fest_agent", want: Retain},
		{name: "meta retained", key: "fest_tracking", want: Retain},
		{name: "non-fest template metadata retained", key: "id", want: Retain},
		{name: "non-fest aliases retained", key: "aliases", want: Retain},
		{name: "approval retained", key: "approval", want: Retain},
		{name: "hooks retained", key: "hooks", want: Retain},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := p.Classify(tc.key); got != tc.want {
				t.Fatalf("Classify(%q) = %s, want %s", tc.key, got, tc.want)
			}
		})
	}
}

func TestV1Tables(t *testing.T) {
	t.Parallel()
	p := V1()
	if p.Version != 1 || Current().Version != Version || Version != 2 {
		t.Fatalf("versions: v1=%d current=%d const=%d", p.Version, Current().Version, Version)
	}
	for n, want := range map[int]bool{1: true, 2: true, 3: false} {
		_, err := ForVersion(n)
		if (err == nil) != want {
			t.Fatalf("ForVersion(%d) err = %v", n, err)
		}
	}
	for _, k := range p.Stripped() {
		if _, dup := p.retain[k]; dup {
			t.Fatalf("%q is in both tables", k)
		}
	}
	if len(p.Retained()) != 30 || len(p.Stripped()) != 3 {
		t.Fatalf("table sizes retain=%d strip=%d, want 30/3 — bump Version if the tables changed on purpose", len(p.Retained()), len(p.Stripped()))
	}
}
