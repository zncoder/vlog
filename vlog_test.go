package vlog

import (
	"bytes"
	"log"
	"reflect"
	"strings"
	"testing"
)

func TestParseFlag(t *testing.T) {
	testcases := []struct {
		in     string
		exact  map[string]Level
		prefix map[string]Level
	}{
		{
			"*=e,foo=i,bar/*=v1,foo/bar/*=v2",
			map[string]Level{
				"foo": info,
			},
			map[string]Level{
				"/":        err,
				"bar/":     v1,
				"foo/bar/": v2,
			},
		},
		{
			"fe=e,fi=i,fv1=v1,fv2=v2",
			map[string]Level{
				"fe":  err,
				"fi":  info,
				"fv1": v1,
				"fv2": v2,
			},
			map[string]Level{},
		},
	}

	for i, tc := range testcases {
		exact, prefix := parseFlag(tc.in)
		if !reflect.DeepEqual(exact, tc.exact) {
			t.Errorf("%d:%v exact: got %v, want %v", i, tc, exact, tc.exact)
		}
		if !reflect.DeepEqual(prefix, tc.prefix) {
			t.Errorf("%d:%v prefix: got %v, want %v", i, tc, prefix, tc.prefix)
		}
	}
}

func clearFilenames() {
	for _, lv := range levelVars {
		lv.File = ""
	}
}

func TestNew(t *testing.T) {
	for _, n := range []string{"", "fa", "fb", "fb/foo", "fc/foo"} {
		newVar(n, "")
	}
	clearFilenames()
	want := []*levelVar{
		&levelVar{},
		&levelVar{Name: "fa"},
		&levelVar{Name: "fb"},
		&levelVar{Name: "fb/foo"},
		&levelVar{Name: "fc/foo"},
	}
	if !reflect.DeepEqual(levelVars, want) {
		t.Errorf("got %v, want %v", levelVars, want)
	}
}

func TestInferName(t *testing.T) {
	testcases := []struct {
		in  string
		out string
	}{
		{
			"/home/foo/src/pa/foo.go",
			"pa",
		},
		{
			"/home/foo/src/pa/foo/bar.go",
			"pa/foo",
		},
		{
			"/home/foo/src/pa/main/foo.go",
			"pa/foo",
		},
		{
			"/home/foo/src/pa/foo/main/bar.go",
			"pa/foo/bar",
		},
	}
	for i, tc := range testcases {
		got := inferName(tc.in)
		if got != tc.out {
			t.Errorf("%d:%s: got %v, want %v", i, tc, got, tc.out)
		}
	}
}

func TestSetLevels(t *testing.T) {
	names := []string{"a", "a/b", "a/b/c", "b/c", "c/d"}
	testcases := []struct {
		in    string
		want  []*levelVar
		print string
	}{
		{
			"*=e,a=i,a/b=v1,c/d/e=i",
			[]*levelVar{
				&levelVar{Level: err},
				&levelVar{Name: "a", Level: info},
				&levelVar{Name: "a/b", Level: v1},
				&levelVar{Name: "a/b/c", Level: err},
				&levelVar{Name: "b/c", Level: err},
				&levelVar{Name: "c/d", Level: err},
			},
			"*=err,a=info,a/b=v1,a/b/c=err,b/c=err,c/d=err",
		},
		{
			"a=i,a/b=v1,a/b/c=v2",
			[]*levelVar{
				&levelVar{},
				&levelVar{Name: "a", Level: info},
				&levelVar{Name: "a/b", Level: v1},
				&levelVar{Name: "a/b/c", Level: v2},
				&levelVar{Name: "b/c", Level: info},
				&levelVar{Name: "c/d", Level: info},
			},
			"*=info,a=info,a/b=v1,a/b/c=v2,b/c=info,c/d=info",
		},
		{
			"a/*=i,a/b/*=v1",
			[]*levelVar{
				&levelVar{},
				&levelVar{Name: "a", Level: info},
				&levelVar{Name: "a/b", Level: v1},
				&levelVar{Name: "a/b/c", Level: v1},
				&levelVar{Name: "b/c", Level: info},
				&levelVar{Name: "c/d", Level: info},
			},
			"*=info,a=info,a/b=v1,a/b/c=v1,b/c=info,c/d=info",
		},
	}
	for i, tc := range testcases {
		levelVars = []*levelVar{&levelVar{}}
		for _, n := range names {
			newVar(n, "")
		}
		setLevels(tc.in)
		clearFilenames()
		if !reflect.DeepEqual(levelVars, tc.want) {
			t.Errorf("%d got=%v, want=%v", i, levelVars, tc.want)
		}
		s := printLevelVars()
		if s != tc.print {
			t.Errorf("%d print: got=%v, want=%v", i, s, tc.print)
		}
	}
}

func TestLog(t *testing.T) {
	levelVars = []*levelVar{&levelVar{}}
	b := new(bytes.Buffer)
	oldlg := lg
	lg = &stderrLogger{log.New(b, "", 0)}
	defer func() {
		lg = oldlg
	}()

	va := newVar("a", "")
	vab := newVar("a/b", "")
	vabc := newVar("a/b/c", "")
	vabd := newVar("a/b/d", "")
	vc := newVar("c", "")
	setLevels("*=e,a=i,a/b/*=v1,a/b/c=v2")
	if *va != info {
		t.Errorf("va got %v, want info", va)
	}
	if *vab != v1 {
		t.Errorf("vab got %v, want v1", vab)
	}
	if *vabc != v2 {
		t.Errorf("vabc got %v, want v2", vabc)
	}
	if *vabd != v1 {
		t.Errorf("vabd got %v, want v1", vabd)
	}
	if *vc != err {
		t.Errorf("vc got %v, want err", vc)
	}

	type fn func(*Level, ...interface{})
	fnE := (*Level).E
	fnI := (*Level).I
	fnV1 := (*Level).V1
	fnV2 := (*Level).V2

	testcases := []struct {
		v  *Level
		fn fn
		m  string
		c  bool
	}{
		{va, fnE, "vaE", true},
		{va, fnI, "vaI", true},
		{va, fnV1, "vaV1", false},
		{vab, fnI, "vabI", true},
		{vab, fnV1, "vabV1", true},
		{vab, fnV2, "vabV2", false},
		{vabc, fnV1, "vabcV1", true},
		{vabc, fnV2, "vabcV2", true},
		{vabd, fnI, "vabdI", true},
		{vabd, fnV1, "vabdV1", true},
		{vabd, fnV2, "vabdV2", false},
		{vc, fnE, "vcE", true},
		{vc, fnI, "vcI", false},
		{vc, fnV1, "vcV1", false},
	}
	for i, tc := range testcases {
		levelVars = []*levelVar{&levelVar{}}
		b.Reset()
		tc.fn(tc.v, tc.m)
		got := b.String()
		if strings.Contains(got, tc.m) != tc.c {
			w := tc.m
			if !tc.c {
				w = ""
			}
			t.Errorf("%d,%v: got %v, want %v", i, tc, got, w)
		}
	}
}

func TestFormat(t *testing.T) {
	b := new(bytes.Buffer)
	oldlg := lg
	lg = &stderrLogger{log.New(b, "", 0)}
	defer func() {
		lg = oldlg
	}()

	testcases := []struct {
		args []interface{}
		want string
	}{
		{
			[]interface{}{"plaintext"},
			"plaintext",
		},
		{
			[]interface{}{"%d is 1", 1},
			"1 is 1",
		},
	}
	var v Level
	for i, tc := range testcases {
		b.Reset()
		v.I(tc.args...)
		got := strings.TrimSpace(b.String())
		if !strings.HasSuffix(got, tc.want) {
			t.Errorf("tc:%d got:%v want:%v", i, got, tc.want)
		}
	}
}

func TestStringer(t *testing.T) {
	called := 0
	f := func() string {
		called++
		return ""
	}

	b := new(bytes.Buffer)
	oldlg := lg
	lg = &stderrLogger{log.New(b, "", 0)}
	defer func() {
		lg = oldlg
	}()

	var v Level
	v.I("info f=%v", Stringer(f))
	if called != 1 {
		t.Errorf("called want:1 got:%d", called)
	}
	v.V1("v1 f=%v", Stringer(f))
	if called != 1 {
		t.Errorf("called want:1 got:%d", called)
	}
}
