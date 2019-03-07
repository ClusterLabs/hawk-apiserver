package main

import (
	"reflect"
	"testing"
)

func TestCommonFunctions(t *testing.T) {
	t.Run("TestIsString", func(t *testing.T) {
		input := "test_string"
		assertTrue(t, IsString(reflect.ValueOf(input)))
	})

	t.Run("TestIsPtr", func(t *testing.T) {
		number := 3
		input := &number
		assertTrue(t, IsPtr(reflect.ValueOf(input)))
	})

	t.Run("TestIsStruct", func(t *testing.T) {
		input := Person{name: "Xin", age: 100}
		assertTrue(t, IsStruct(reflect.ValueOf(input)))
	})

	t.Run("TestIsSlice", func(t *testing.T) {
		input := make([]string, 0)
		assertTrue(t, IsSlice(reflect.ValueOf(input)))
	})

	t.Run("TestIsBlank", func(t *testing.T) {
		var crmMon *CrmMon
		assertTrue(t, IsBlank(reflect.ValueOf(crmMon)))
	})
}

func TestGetNumField(t *testing.T) {
	allTests := []struct {
		Name string
		Input interface{}
		Want int
	}{
		{
			"Blank",
			Person{},
			0,
		},
		{
			"Struct",
			Person{"Xin", 100},
			2,
		},
		{
			"Ptr",
			&Person{"Xin", 100},
			2,
		},
		{
			"Slice",
			[]Person {
				{"Tom", 1},
				{"Jake", 21},
				{"Room", 33},
			},
			3,
		},
	}

	for _, test := range allTests {
		t.Run(test.Name, func(t *testing.T) {
			got := GetNumField(test.Input)
			if got != test.Want {
				t.Errorf("got %d, want %d", got, test.Want)
			}
		})
	}
}

type Person struct {
	name string
	age  int
}

func assertTrue(t *testing.T, got bool) {
	t.Helper()
	if got != true {
		t.Errorf("got %t, want true", got)
	}
}

func TestFetchNv(t *testing.T) {
	t.Run("TestExtractMetas", func(t *testing.T) {
		nv_list := []*Nvpair {
			&Nvpair{Name: "target-role", Value: "Stopped"},
			&Nvpair{Name: "description", Value: "test"},
		}
		metas := []*MetaAttributes {
			&MetaAttributes{Nvpair: nv_list},
		}
		res := FetchNv(metas)

		assertTrue(t, IsMap(reflect.ValueOf(res)))
		assertEqualString(t, res["description"], "test")
		assertEqualString(t, res["target-role"], "Stopped")
	})

	t.Run("TestExtractOpList", func(t *testing.T) {
		op_list := []Op {
			{
				Id:		"op-monitor-10s",
				Name:		"monitor",
				Interval:	"10s",
				Timeout:	"20s",
			},
			{
				Id:		"op-start-0",
				Name:		"start",
				Timeout:	"20s",
				OnFail:		"test",
			},
		}
		res := FetchNv2(op_list[0])
		assertTrue(t, IsMap(reflect.ValueOf(res)))
		assertEqualString(t, res["id"].(string), "op-monitor-10s")

		res = FetchNv2(op_list[1])
		assertTrue(t, IsMap(reflect.ValueOf(res)))
		assertEqualString(t, res["on-fail"].(string), "test")
	})
}

func assertEqualString(t *testing.T, got string, want string) {
	t.Helper()
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
