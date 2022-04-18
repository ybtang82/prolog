package prolog

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ichiban/prolog/engine"
)

func TestNew(t *testing.T) {
	i := New(nil, nil)
	assert.NotNil(t, i)

	t.Run("number_chars", func(t *testing.T) {
		// http://www.complang.tuwien.ac.at/ulrich/iso-prolog/number_chars
		p := New(nil, nil)

		// Section 0
		assert.NoError(t, p.QuerySolution(`number_chars(1.2,['1',.,'2']).`).Err())
		assert.NoError(t, p.QuerySolution(`number_chars(1.0e9,['1',.,'0','E','9']).`).Err())
		assert.NoError(t, p.QuerySolution(`number_chars(1,['0','1']).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(1,[a]), error(syntax_error(_), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(1,[]), error(syntax_error(_), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(1,[[]]), error(type_error(character,[]), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(1,[' ',[]]), error(type_error(character,[]), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(1,[0]), error(type_error(character,0), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(1,[_,[]]), error(type_error(character,[]), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,[X]), error(instantiation_error,_), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,['0'|_]), error(instantiation_error,_), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,'1'), error(type_error(list,'1'), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,[a|a]), error(type_error(list,[a|a]), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,[49]), error(type_error(character,49), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,[]), error(syntax_error(_), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,['3',' ']), error(syntax_error(_), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,['3',.]), error(syntax_error(_), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`number_chars(N,[' ','1']), N = 1.`).Err())
		assert.NoError(t, p.QuerySolution(`number_chars(N,['\n','1']), N = 1.`).Err())
		assert.NoError(t, p.QuerySolution(`number_chars(N,[' ','0','''',a]), N = 97.`).Err())
		assert.NoError(t, p.QuerySolution(`number_chars(N,[-,' ','1']), N = -1.`).Err())
		assert.NoError(t, p.QuerySolution(`number_chars(N,[/,*,*,/,'1']), N = 1.`).Err())
		assert.NoError(t, p.QuerySolution(`number_chars(N,['%','\n','1']), N = 1.`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,[-,/,*,*,/,'1']), error(syntax_error(_), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,['1',e,'1']), error(syntax_error(_), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,['1',.,'0',e]), error(syntax_error(_), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,['1',.,'0',e,e]), error(syntax_error(_), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`number_chars(N,['0',x,'1']), N = 1.`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,['0','X','1']), error(syntax_error(_), _), true).`).Err())
		assert.Error(t, p.QuerySolution(`catch(number_chars(1E1,_), error(syntax_error(_), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`\+number_chars(1,['.'|_]).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,[+,'1']), error(syntax_error(_), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,[+,' ','1']), error(syntax_error(_), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,['''',+,'''','1']), error(syntax_error(_), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,['11']), error(type_error(character,_), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,['1.1']), error(type_error(character,_), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(1+1,['2']), error(type_error(number,1+1), _), true).`).Err())

		// Section 2
		assert.NoError(t, p.QuerySolution(`number_chars(1,[C]), C = '1'.`).Err())
		assert.NoError(t, p.QuerySolution(`\+number_chars(1,[C,D]).`).Err())
		assert.NoError(t, p.QuerySolution(`\+number_chars(1,[C,C]).`).Err())
		assert.NoError(t, p.QuerySolution(`\+number_chars(0,[C,C]).`).Err())
		assert.NoError(t, p.QuerySolution(`number_chars(10,[C,D]), C = '1', D = '0'.`).Err())
		assert.NoError(t, p.QuerySolution(`\+number_chars(100,[C,D]).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,[X|2]), error(instantiation_error,_), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(N,[1|_]), error(type_error(character,1), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(V,[1|2]), error(type_error(character,1), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars([],1), error(type_error(list,1), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(1,1), error(type_error(list,1), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(1,[a|2]), error(type_error(list,[a|2]), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(1,[_|2]), error(type_error(list,[_|2]), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(1,[[]|_]), error(type_error(character,[]), _), true).`).Err())
		assert.NoError(t, p.QuerySolution(`catch(number_chars(1,[[]|2]), error(type_error(character,[]), _), true).`).Err())

		assert.NoError(t, p.QuerySolution(`catch((L=['1'|L], number_chars(N,L)), error(type_error(list,['1'|_]), _), L=['1'|L]).`).Err())
	})
}

func TestInterpreter_Exec(t *testing.T) {
	t.Run("fact", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			var i Interpreter
			assert.NoError(t, i.Exec(`append(nil, L, L).`))
		})

		t.Run("not callable", func(t *testing.T) {
			var i Interpreter
			assert.Error(t, i.Exec(`0.`))
		})
	})

	t.Run("rule", func(t *testing.T) {
		var i Interpreter
		i.Register3("op", i.Op)
		assert.NoError(t, i.Exec(":-(op(1200, xfx, :-))."))
		assert.NoError(t, i.Exec(`append(cons(X, L1), L2, cons(X, L3)) :- append(L1, L2, L3).`))
	})

	t.Run("bindvars", func(t *testing.T) {
		var i Interpreter
		assert.NoError(t, i.Exec("foo(?, ?, ?, ?).", "a", 1, 2.0, []string{"abc", "def"}))
	})

	t.Run("shebang", func(t *testing.T) {
		t.Run("multiple lines", func(t *testing.T) {
			var i Interpreter
			assert.NoError(t, i.Exec(`#!/usr/bin/env 1pl
append(nil, L, L).`))
		})

		t.Run("only shebang line", func(t *testing.T) {
			var i Interpreter
			assert.Equal(t, engine.ErrInsufficient, i.Exec(`#!/usr/bin/env 1pl`))
		})
	})

	t.Run("consult", func(t *testing.T) {
		i := New(nil, nil)

		t.Run("variable", func(t *testing.T) {
			assert.Error(t, i.Exec(":- consult(X)."))
		})

		t.Run("non-proper list", func(t *testing.T) {
			assert.Error(t, i.Exec(":- consult([?|_]).", "testdata/empty.txt"))
		})

		t.Run("proper list", func(t *testing.T) {
			t.Run("ok", func(t *testing.T) {
				assert.NoError(t, i.Exec(":- consult([])."))
				assert.NoError(t, i.Exec(":- consult([?]).", "testdata/empty.txt"))
				assert.NoError(t, i.Exec(":- consult(?).", []string{
					"testdata/empty.txt",
					"testdata/empty.txt",
				}))
			})

			t.Run("variable", func(t *testing.T) {
				assert.Error(t, i.Exec(":- consult([X])."))
			})

			t.Run("not an atom", func(t *testing.T) {
				assert.Error(t, i.Exec(":- consult([1])."))
			})

			t.Run("invalid", func(t *testing.T) {
				assert.Error(t, i.Exec(":- consult([?]).", "testdata/abc.txt"))
			})

			t.Run("not found", func(t *testing.T) {
				assert.Error(t, i.Exec(":- consult([?]).", "testdata/not_found.txt"))
			})
		})

		t.Run("atom", func(t *testing.T) {
			t.Run("ok", func(t *testing.T) {
				assert.NoError(t, i.Exec(":- consult(?).", "testdata/empty.txt"))
			})

			t.Run("ng", func(t *testing.T) {
				assert.Error(t, i.Exec(":- consult(?).", "testdata/abc.txt"))
			})
		})

		t.Run("compound", func(t *testing.T) {
			assert.Error(t, i.Exec(":- consult(foo(bar))."))
		})

		t.Run("not an atom ", func(t *testing.T) {
			assert.Error(t, i.Exec(":- consult(1)."))
		})
	})

	t.Run("term_expansion/2 throws an exception", func(t *testing.T) {
		i := New(nil, nil)
		assert.NoError(t, i.Exec(`term_expansion(_, _) :- throw(fail).`))

		assert.Error(t, i.Exec("a."))
	})
}

func TestInterpreter_Query(t *testing.T) {
	var i Interpreter
	i.Register3("op", i.Op)
	assert.NoError(t, i.Exec(":-(op(1200, xfx, :-))."))
	assert.NoError(t, i.Exec("append(nil, L, L)."))
	assert.NoError(t, i.Exec("append(cons(X, L1), L2, cons(X, L3)) :- append(L1, L2, L3)."))

	t.Run("fact", func(t *testing.T) {
		sols, err := i.Query(`append(X, Y, Z).`)
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, sols.Close())
		}()

		m := map[string]engine.Term{}

		assert.True(t, sols.Next())
		assert.NoError(t, sols.Scan(m))
		assert.Len(t, m, 3)
		assert.Equal(t, engine.Atom("nil"), m["X"])
		assert.Equal(t, engine.Variable("Z"), m["Y"])
		assert.Equal(t, engine.Variable("Z"), m["Z"])
	})

	t.Run("rule", func(t *testing.T) {
		sols, err := i.Query(`append(cons(a, cons(b, nil)), cons(c, nil), X).`)
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, sols.Close())
		}()

		m := map[string]engine.Term{}

		assert.True(t, sols.Next())
		assert.NoError(t, sols.Scan(m))
		assert.Equal(t, map[string]engine.Term{
			"X": &engine.Compound{
				Functor: "cons",
				Args: []engine.Term{
					engine.Atom("a"),
					&engine.Compound{
						Functor: "cons",
						Args: []engine.Term{
							engine.Atom("b"),
							&engine.Compound{
								Functor: "cons",
								Args:    []engine.Term{engine.Atom("c"), engine.Atom("nil")},
							},
						},
					},
				},
			},
		}, m)
	})

	t.Run("bindvars", func(t *testing.T) {
		var i Interpreter
		assert.NoError(t, i.Exec("foo(a, 1, 2.0, [abc, def])."))

		sols, err := i.Query(`foo(?, ?, ?, ?).`, "a", 1, 2.0, []string{"abc", "def"})
		assert.NoError(t, err)

		m := map[string]interface{}{}

		assert.True(t, sols.Next())
		assert.NoError(t, sols.Scan(m))
		assert.Equal(t, map[string]interface{}{}, m)
	})

	t.Run("scan to struct", func(t *testing.T) {
		var i Interpreter
		assert.NoError(t, i.Exec("foo(a, 1, 2.0, [abc, def])."))

		sols, err := i.Query(`foo(A, B, C, D).`)
		assert.NoError(t, err)

		type result struct {
			A    string
			B    int
			C    float64
			List []string `prolog:"D"`
		}

		assert.True(t, sols.Next())

		var r result
		assert.NoError(t, sols.Scan(&r))
		assert.Equal(t, result{
			A:    "a",
			B:    1,
			C:    2.0,
			List: []string{"abc", "def"},
		}, r)
	})
}

func TestMisc(t *testing.T) {
	t.Run("negation", func(t *testing.T) {
		i := New(nil, nil)
		sols, err := i.Query(`\+true.`)
		assert.NoError(t, err)

		assert.False(t, sols.Next())
	})

	t.Run("cut", func(t *testing.T) {
		// https://www.cs.uleth.ca/~gaur/post/prolog-cut-negation/
		t.Run("p", func(t *testing.T) {
			i := New(nil, nil)
			assert.NoError(t, i.Exec(`
p(a).
p(b):-!.
p(c).
`))

			t.Run("single", func(t *testing.T) {
				sols, err := i.Query(`p(X).`)
				assert.NoError(t, err)
				defer sols.Close()

				var s struct {
					X string
				}

				assert.True(t, sols.Next())
				assert.NoError(t, sols.Scan(&s))
				assert.Equal(t, "a", s.X)

				assert.True(t, sols.Next())
				assert.NoError(t, sols.Scan(&s))
				assert.Equal(t, "b", s.X)

				assert.False(t, sols.Next())
			})

			t.Run("double", func(t *testing.T) {
				sols, err := i.Query(`p(X), p(Y).`)
				assert.NoError(t, err)
				defer sols.Close()

				var s struct {
					X string
					Y string
				}

				assert.True(t, sols.Next())
				assert.NoError(t, sols.Scan(&s))
				assert.Equal(t, "a", s.X)
				assert.Equal(t, "a", s.Y)

				assert.True(t, sols.Next())
				assert.NoError(t, sols.Scan(&s))
				assert.Equal(t, "a", s.X)
				assert.Equal(t, "b", s.Y)

				assert.True(t, sols.Next())
				assert.NoError(t, sols.Scan(&s))
				assert.Equal(t, "b", s.X)
				assert.Equal(t, "a", s.Y)

				assert.True(t, sols.Next())
				assert.NoError(t, sols.Scan(&s))
				assert.Equal(t, "b", s.X)
				assert.Equal(t, "b", s.Y)

				assert.False(t, sols.Next())
			})
		})

		// http://www.cse.unsw.edu.au/~billw/dictionaries/prolog/cut.html
		t.Run("teaches", func(t *testing.T) {
			i := New(nil, nil)
			assert.NoError(t, i.Exec(`
teaches(dr_fred, history).
teaches(dr_fred, english).
teaches(dr_fred, drama).
teaches(dr_fiona, physics).
studies(alice, english).
studies(angus, english).
studies(amelia, drama).
studies(alex, physics).
`))

			t.Run("without cut", func(t *testing.T) {
				sols, err := i.Query(`teaches(dr_fred, Course), studies(Student, Course).`)
				assert.NoError(t, err)
				defer func() {
					assert.NoError(t, sols.Close())
				}()

				type cs struct {
					Course  string
					Student string
				}
				var s cs

				assert.True(t, sols.Next())
				assert.NoError(t, sols.Scan(&s))
				assert.Equal(t, cs{
					Course:  "english",
					Student: "alice",
				}, s)

				assert.True(t, sols.Next())
				assert.NoError(t, sols.Scan(&s))
				assert.Equal(t, cs{
					Course:  "english",
					Student: "angus",
				}, s)

				assert.True(t, sols.Next())
				assert.NoError(t, sols.Scan(&s))
				assert.Equal(t, cs{
					Course:  "drama",
					Student: "amelia",
				}, s)

				assert.False(t, sols.Next())
			})

			t.Run("with cut in the middle", func(t *testing.T) {
				sols, err := i.Query(`teaches(dr_fred, Course), !, studies(Student, Course).`)
				assert.NoError(t, err)
				defer func() {
					assert.NoError(t, sols.Close())
				}()

				assert.False(t, sols.Next())
			})

			t.Run("with cut at the end", func(t *testing.T) {
				sols, err := i.Query(`teaches(dr_fred, Course), studies(Student, Course), !.`)
				assert.NoError(t, err)
				defer func() {
					assert.NoError(t, sols.Close())
				}()

				type cs struct {
					Course  string
					Student string
				}
				var s cs

				assert.True(t, sols.Next())
				assert.NoError(t, sols.Scan(&s))
				assert.Equal(t, cs{
					Course:  "english",
					Student: "alice",
				}, s)

				assert.False(t, sols.Next())
			})

			t.Run("with cut at the beginning", func(t *testing.T) {
				sols, err := i.Query(`!, teaches(dr_fred, Course), studies(Student, Course).`)
				assert.NoError(t, err)
				defer func() {
					assert.NoError(t, sols.Close())
				}()

				type cs struct {
					Course  string
					Student string
				}
				var s cs

				assert.True(t, sols.Next())
				assert.NoError(t, sols.Scan(&s))
				assert.Equal(t, cs{
					Course:  "english",
					Student: "alice",
				}, s)

				assert.True(t, sols.Next())
				assert.NoError(t, sols.Scan(&s))
				assert.Equal(t, cs{
					Course:  "english",
					Student: "angus",
				}, s)

				assert.True(t, sols.Next())
				assert.NoError(t, sols.Scan(&s))
				assert.Equal(t, cs{
					Course:  "drama",
					Student: "amelia",
				}, s)

				assert.False(t, sols.Next())
			})
		})

		t.Run("call/1 makes a difference", func(t *testing.T) {
			t.Run("with", func(t *testing.T) {
				i := New(nil, nil)
				sols, err := i.Query(`call(!), fail; true.`)
				assert.NoError(t, err)
				defer sols.Close()

				assert.True(t, sols.Next())
			})

			t.Run("without", func(t *testing.T) {
				i := New(nil, nil)
				sols, err := i.Query(`!, fail; true.`)
				assert.NoError(t, err)
				defer sols.Close()

				assert.False(t, sols.Next())
			})
		})
	})

	t.Run("repeat", func(t *testing.T) {
		t.Run("cut", func(t *testing.T) {
			i := New(nil, nil)
			sols, err := i.Query("repeat, !, fail.")
			assert.NoError(t, err)
			assert.False(t, sols.Next())
		})

		t.Run("stream", func(t *testing.T) {
			i := New(nil, nil)
			sols, err := i.Query("repeat, (X = a; X = b).")
			assert.NoError(t, err)

			var s struct {
				X string
			}

			assert.True(t, sols.Next())
			assert.NoError(t, sols.Scan(&s))
			assert.Equal(t, "a", s.X)

			assert.True(t, sols.Next())
			assert.NoError(t, sols.Scan(&s))
			assert.Equal(t, "b", s.X)

			assert.True(t, sols.Next())
			assert.NoError(t, sols.Scan(&s))
			assert.Equal(t, "a", s.X)

			assert.True(t, sols.Next())
			assert.NoError(t, sols.Scan(&s))
			assert.Equal(t, "b", s.X)
		})
	})

	t.Run("atom_chars", func(t *testing.T) {
		i := New(nil, nil)
		sols, err := i.Query("atom_chars(f(a), L).")
		assert.NoError(t, err)
		assert.False(t, sols.Next())
	})

	t.Run("term_eq", func(t *testing.T) {
		i := New(nil, nil)
		sols, err := i.Query("f(a) == f(a).")
		assert.NoError(t, err)
		assert.True(t, sols.Next())
	})

	t.Run("call cut", func(t *testing.T) {
		i := New(nil, nil)
		assert.NoError(t, i.Exec("foo :- call(true), !."))
		assert.NoError(t, i.Exec("foo :- throw(unreachable)."))
		sols, err := i.Query("foo.")
		assert.NoError(t, err)
		assert.True(t, sols.Next())
		assert.False(t, sols.Next())
		assert.NoError(t, sols.Err())
	})

	t.Run("catch cut", func(t *testing.T) {
		i := New(nil, nil)
		assert.NoError(t, i.Exec("foo :- catch(true, _, true), !."))
		assert.NoError(t, i.Exec("foo :- throw(unreachable)."))
		sols, err := i.Query("foo.")
		assert.NoError(t, err)
		assert.True(t, sols.Next())
		assert.False(t, sols.Next())
		assert.NoError(t, sols.Err())
	})

	t.Run("counter", func(t *testing.T) {
		i := New(nil, nil)
		assert.NoError(t, i.Exec(":- dynamic(count/1)."))
		assert.NoError(t, i.Exec("count(0)."))
		assert.NoError(t, i.Exec("next(N) :- retract(count(X)), N is X + 1, asserta(count(N))."))

		var s struct {
			X int
		}

		sols, err := i.Query("next(X).")
		assert.NoError(t, err)
		assert.True(t, sols.Next())
		assert.NoError(t, sols.Scan(&s))
		assert.Equal(t, 1, s.X)
		assert.False(t, sols.Next())
		assert.NoError(t, sols.Err())
		assert.NoError(t, sols.Close())

		sols, err = i.Query("next(X).")
		assert.NoError(t, err)
		assert.True(t, sols.Next())
		assert.NoError(t, sols.Scan(&s))
		assert.Equal(t, 2, s.X)
		assert.False(t, sols.Next())
		assert.NoError(t, sols.Err())
		assert.NoError(t, sols.Close())

		sols, err = i.Query("next(X).")
		assert.NoError(t, err)
		assert.True(t, sols.Next())
		assert.NoError(t, sols.Scan(&s))
		assert.Equal(t, 3, s.X)
		assert.False(t, sols.Next())
		assert.NoError(t, sols.Err())
		assert.NoError(t, sols.Close())
	})
}

func TestInterpreter_QuerySolution(t *testing.T) {
	var i Interpreter
	assert.NoError(t, i.Exec(`
foo(a, b).
foo(b, c).
foo(c, d).
`))

	t.Run("ok", func(t *testing.T) {
		t.Run("struct", func(t *testing.T) {
			sol := i.QuerySolution(`foo(X, Y).`)

			var s struct {
				X   string
				Foo string `prolog:"Y"`
			}
			assert.NoError(t, sol.Scan(&s))
			assert.Equal(t, "a", s.X)
			assert.Equal(t, "b", s.Foo)
		})

		t.Run("map", func(t *testing.T) {
			sol := i.QuerySolution(`foo(X, Y).`)

			m := map[string]string{}
			assert.NoError(t, sol.Scan(m))
			assert.Equal(t, []string{"X", "Y"}, sol.Vars())
			assert.Equal(t, "a", m["X"])
			assert.Equal(t, "b", m["Y"])
		})
	})

	t.Run("invalid query", func(t *testing.T) {
		sol := i.QuerySolution(``)
		assert.Error(t, sol.Err())
	})

	t.Run("no solutions", func(t *testing.T) {
		sol := i.QuerySolution(`foo(e, f).`)
		assert.Equal(t, ErrNoSolutions, sol.Err())
		assert.Empty(t, sol.Vars())
	})

	t.Run("runtime error", func(t *testing.T) {
		err := errors.New("something went wrong")

		i.Register0("error", func(k func(*engine.Env) *engine.Promise, env *engine.Env) *engine.Promise {
			return engine.Error(err)
		})
		sol := i.QuerySolution(`error.`)
		assert.Equal(t, err, sol.Err())

		var s struct{}
		assert.Error(t, sol.Scan(&s))
	})
}

func ExampleInterpreter_Exec_placeholders() {
	p := New(nil, os.Stdout)

	_ = p.Exec(`my_atom(?).`, "foo")
	sols, _ := p.Query(`my_atom(A), atom(A), write(A), nl.`)
	sols.Next()

	_ = p.Exec(`my_int(?, ?, ?, ?, ?).`, int8(1), int16(1), int32(1), int64(1), 1)
	sols, _ = p.Query(`my_int(I, I, I, I, I), integer(I), write(I), nl.`)
	sols.Next()

	_ = p.Exec(`my_float(?, ?).`, float32(1), float64(1))
	sols, _ = p.Query(`my_float(F, F), float(F), write(F), nl.`)
	sols.Next()

	_ = p.Exec(`my_atom_list(?).`, []string{"foo", "bar", "baz"})
	sols, _ = p.Query(`my_atom_list(As), maplist(atom, As), write(As), nl.`)
	sols.Next()

	_ = p.Exec(`my_int_list(?).`, []int{1, 2, 3})
	sols, _ = p.Query(`my_int_list(Is), maplist(integer, Is), write(Is), nl.`)
	sols.Next()

	_ = p.Exec(`my_float_list(?).`, []float64{1, 2, 3})
	sols, _ = p.Query(`my_float_list(Fs), maplist(float, Fs), write(Fs), nl.`)
	sols.Next()

	// Output:
	// foo
	// 1
	// 1.0
	// [foo, bar, baz]
	// [1, 2, 3]
	// [1.0, 2.0, 3.0]
}

func ExampleInterpreter_Query_placeholders() {
	p := New(nil, os.Stdout)
	sols, _ := p.Query(`A = ?, atom(A), write(A), nl.`, "foo")
	sols.Next()
	sols, _ = p.Query(`(I, I, I, I, I) = (?, ?, ?, ?, ?), integer(I), write(I), nl.`, int8(1), int16(1), int32(1), int64(1), 1)
	sols.Next()
	sols, _ = p.Query(`(F, F) = (?, ?), float(F), write(F), nl.`, float32(1), float64(1))
	sols.Next()
	sols, _ = p.Query(`L = ?, maplist(atom, L), write(L), nl.`, []string{"foo", "bar", "baz"})
	sols.Next()
	sols, _ = p.Query(`L = ?, maplist(integer, L), write(L), nl.`, []int{1, 2, 3})
	sols.Next()
	sols, _ = p.Query(`L = ?, maplist(float, L), write(L), nl.`, []float64{1, 2, 3})
	sols.Next()

	// Output:
	// foo
	// 1
	// 1.0
	// [foo, bar, baz]
	// [1, 2, 3]
	// [1.0, 2.0, 3.0]
}

func ExampleInterpreter_New_phrase() {
	p := New(nil, nil)
	_ = p.Exec(`
determiner --> [the].
determiner --> [a].

noun --> [boy].
noun --> [girl].

verb --> [likes].
verb --> [scares].

noun_phrase --> determiner, noun.
noun_phrase --> noun.

verb_phrase --> verb.
verb_phrase --> verb, noun_phrase.

sentence --> noun_phrase, verb_phrase.
`)

	sols, _ := p.Query(`phrase([the], [the]).`)
	fmt.Printf("%t\n", sols.Next())

	sols, _ = p.Query(`phrase(sentence, [the, girl, likes, the, boy]).`)
	fmt.Printf("%t\n", sols.Next())

	sols, _ = p.Query(`phrase(sentence, [the, girl, likes, the, boy, today]).`)
	fmt.Printf("%t\n", sols.Next())

	sols, _ = p.Query(`phrase(sentence, [the, girl, likes]).`)
	fmt.Printf("%t\n", sols.Next())

	sols, _ = p.Query(`phrase(sentence, Sentence).`)
	for sols.Next() {
		var s struct {
			Sentence []string
		}
		_ = sols.Scan(&s)
		fmt.Printf("Sentence = %s\n", s.Sentence)
		break // Many other sentences follow.
	}

	sols, _ = p.Query(`phrase(noun_phrase, [the, girl, scares, the, boy], Rest).`)
	for sols.Next() {
		var s struct {
			Rest []string
		}
		_ = sols.Scan(&s)
		fmt.Printf("Rest = %s\n", s.Rest)
	}

	// Output:
	// true
	// true
	// false
	// true
	// Sentence = [the boy likes]
	// Rest = [scares the boy]
}

func ExampleInterpreter_New_subsumes_term() {
	p := New(nil, nil)

	sols, _ := p.Query(`subsumes_term(a, a).`)
	fmt.Printf("%t\n", sols.Next())

	sols, _ = p.Query(`subsumes_term(f(X,Y), f(Z,Z)).`)
	fmt.Printf("%t\n", sols.Next())

	sols, _ = p.Query(`subsumes_term(f(Z,Z), f(X,Y)).`)
	fmt.Printf("%t\n", sols.Next())

	sols, _ = p.Query(`subsumes_term(g(X), g(f(X))).`)
	fmt.Printf("%t\n", sols.Next())

	sols, _ = p.Query(`subsumes_term(X, f(X)).`)
	fmt.Printf("%t\n", sols.Next())

	sols, _ = p.Query(`subsumes_term(X, Y), subsumes_term(Y, f(X)).`)
	fmt.Printf("%t\n", sols.Next())

	// Output:
	// true
	// true
	// false
	// false
	// false
	// true
}

func ExampleInterpreter_New_callable() {
	p := New(nil, nil)

	sols, _ := p.Query(`callable(a).`)
	fmt.Printf("%t\n", sols.Next())

	sols, _ = p.Query(`callable(3).`)
	fmt.Printf("%t\n", sols.Next())

	sols, _ = p.Query(`callable(X).`)
	fmt.Printf("%t\n", sols.Next())

	sols, _ = p.Query(`callable((1,2)).`)
	fmt.Printf("%t\n", sols.Next())

	// Output:
	// true
	// false
	// false
	// true
}

func ExampleInterpreter_New_acyclicTerm() {
	p := New(nil, nil)

	sols, _ := p.Query(`acyclic_term(a(1, _)).`)
	fmt.Printf("%t\n", sols.Next())

	sols, _ = p.Query(`X = f(X), acyclic_term(X).`)
	fmt.Printf("%t\n", sols.Next())

	// Output:
	// true
	// false
}

func ExampleInterpreter_New_ground() {
	p := New(nil, nil)

	sols, _ := p.Query(`ground(3).`)
	fmt.Printf("%t\n", sols.Next())

	sols, _ = p.Query(`ground(a(1, _)).`)
	fmt.Printf("%t\n", sols.Next())

	// Output:
	// true
	// false
}

func ExampleInterpreter_New_sort() {
	p := New(nil, nil)

	sols, _ := p.Query(`sort([1, 1], Sorted).`)
	for sols.Next() {
		var s struct {
			Sorted []int
		}
		_ = sols.Scan(&s)
		fmt.Printf("Sorted = %d\n", s.Sorted)
	}

	sols, _ = p.Query(`sort([X, 1], [1, 1]).`)
	for sols.Next() {
		var s struct {
			X int
		}
		_ = sols.Scan(&s)
		fmt.Printf("X = %d\n", s.X)
	}

	sols, _ = p.Query(`sort([1, 1], [1, 1]).`)
	fmt.Printf("%t\n", sols.Next())

	sols, _ = p.Query(`sort([V], V).`)
	fmt.Printf("%t\n", sols.Next())

	sols, _ = p.Query(`sort([f(U),U,U,f(V),f(U),V],L).`)
	fmt.Printf("%t\n", sols.Next())

	// Output:
	// Sorted = [1]
	// X = 1
	// false
	// true
	// true
}
