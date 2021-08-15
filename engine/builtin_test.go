package engine

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"
)

func TestVM_Call(t *testing.T) {
	var vm VM

	t.Run("undefined atom", func(t *testing.T) {
		env := Env{}
		ok, err := vm.Call(Atom("foo"), Success, &env).Force()
		assert.Equal(t, existenceErrorProcedure(&Compound{
			Functor: "/",
			Args:    []Term{Atom("foo"), Integer(0)},
		}), err)
		assert.False(t, ok)
	})

	vm.procedures = map[procedureIndicator]procedure{{name: "foo", arity: 0}: clauses{}}

	t.Run("defined atom", func(t *testing.T) {
		env := Env{}
		ok, err := vm.Call(Atom("foo"), Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("undefined compound", func(t *testing.T) {
		ok, err := vm.Call(&Compound{Functor: "bar", Args: []Term{NewVariable(), NewVariable()}}, Success, &Env{}).Force()
		assert.Equal(t, existenceErrorProcedure(&Compound{
			Functor: "/",
			Args:    []Term{Atom("bar"), Integer(2)},
		}), err)
		assert.False(t, ok)
	})

	vm.procedures = map[procedureIndicator]procedure{{name: "bar", arity: 2}: clauses{}}

	t.Run("defined compound", func(t *testing.T) {
		ok, err := vm.Call(&Compound{Functor: "bar", Args: []Term{NewVariable(), NewVariable()}}, Success, &Env{}).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("variable", func(t *testing.T) {
		env := Env{}
		x := Variable("X")

		ok, err := vm.Call(x, Success, &env).Force()
		assert.Equal(t, instantiationError(x), err)
		assert.False(t, ok)
	})

	t.Run("not callable", func(t *testing.T) {
		env := Env{}
		ok, err := vm.Call(Integer(0), Success, &env).Force()
		assert.Equal(t, typeErrorCallable(Integer(0)), err)
		assert.False(t, ok)
	})
}

func TestUnify(t *testing.T) {
	t.Run("unifiable", func(t *testing.T) {
		env := Env{}
		x := Variable("X")
		ok, err := Unify(x, &Compound{
			Functor: "f",
			Args:    []Term{Atom("a")},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		assert.Equal(t, &Compound{
			Functor: "f",
			Args:    []Term{Atom("a")},
		}, env.Resolve(x))
	})

	t.Run("not unifiable", func(t *testing.T) {
		env := Env{}
		ok, err := Unify(Atom("a"), &Compound{
			Functor: "f",
			Args:    []Term{Atom("a")},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("loop", func(t *testing.T) {
		env := Env{}
		x := Variable("X")
		ok, err := Unify(x, &Compound{
			Functor: "f",
			Args:    []Term{x},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})
}

func TestUnifyWithOccursCheck(t *testing.T) {
	t.Run("unifiable", func(t *testing.T) {
		env := Env{}
		x := Variable("X")
		ok, err := UnifyWithOccursCheck(x, &Compound{
			Functor: "f",
			Args:    []Term{Atom("a")},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		assert.Equal(t, &Compound{
			Functor: "f",
			Args:    []Term{Atom("a")},
		}, env.Resolve(x))
	})

	t.Run("not unifiable", func(t *testing.T) {
		env := Env{}
		ok, err := UnifyWithOccursCheck(Atom("a"), &Compound{
			Functor: "f",
			Args:    []Term{Atom("a")},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("loop", func(t *testing.T) {
		env := Env{}
		x := Variable("X")
		ok, err := UnifyWithOccursCheck(x, &Compound{
			Functor: "f",
			Args:    []Term{x},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestTypeVar(t *testing.T) {
	t.Run("var", func(t *testing.T) {
		env := Env{}
		ok, err := TypeVar(NewVariable(), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("not var", func(t *testing.T) {
		env := Env{}
		ok, err := TypeVar(Atom("foo"), Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestTypeFloat(t *testing.T) {
	t.Run("float", func(t *testing.T) {
		env := Env{}
		ok, err := TypeFloat(Float(1.0), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("not float", func(t *testing.T) {
		env := Env{}
		ok, err := TypeFloat(Atom("foo"), Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestTypeInteger(t *testing.T) {
	t.Run("integer", func(t *testing.T) {
		env := Env{}
		ok, err := TypeInteger(Integer(1), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("not integer", func(t *testing.T) {
		env := Env{}
		ok, err := TypeInteger(Atom("foo"), Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestTypeAtom(t *testing.T) {
	t.Run("atom", func(t *testing.T) {
		env := Env{}
		ok, err := TypeAtom(Atom("foo"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("not atom", func(t *testing.T) {
		env := Env{}
		ok, err := TypeAtom(Integer(1), Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestTypeCompound(t *testing.T) {
	t.Run("compound", func(t *testing.T) {
		env := Env{}
		ok, err := TypeCompound(&Compound{
			Functor: "foo",
			Args:    []Term{Atom("a")},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("not compound", func(t *testing.T) {
		env := Env{}
		ok, err := TypeCompound(Atom("foo"), Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestFunctor(t *testing.T) {
	t.Run("term is instantiated", func(t *testing.T) {
		t.Run("float", func(t *testing.T) {
			env := Env{}
			name, arity := Variable("Name"), Variable("Arity")
			ok, err := Functor(Float(2.0), name, arity, func(env Env) Promise {
				assert.Equal(t, Float(2.0), env.Resolve(name))
				assert.Equal(t, Integer(0), env.Resolve(arity))
				return Bool(true)
			}, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})

		t.Run("integer", func(t *testing.T) {
			env := Env{}
			name, arity := NewVariable(), NewVariable()
			ok, err := Functor(Integer(2), name, arity, func(env Env) Promise {
				assert.Equal(t, Integer(2), env.Resolve(name))
				assert.Equal(t, Integer(0), env.Resolve(arity))
				return Bool(true)
			}, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})

		t.Run("atom", func(t *testing.T) {
			env := Env{}
			name, arity := NewVariable(), NewVariable()
			ok, err := Functor(Atom("foo"), name, arity, func(env Env) Promise {
				assert.Equal(t, Atom("foo"), env.Resolve(name))
				assert.Equal(t, Integer(0), env.Resolve(arity))
				return Bool(true)
			}, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})

		t.Run("compound", func(t *testing.T) {
			env := Env{}
			name, arity := NewVariable(), NewVariable()
			ok, err := Functor(&Compound{
				Functor: "f",
				Args:    []Term{Atom("a"), Atom("b"), Atom("c")},
			}, name, arity, func(env Env) Promise {
				assert.Equal(t, Atom("f"), env.Resolve(name))
				assert.Equal(t, Integer(3), env.Resolve(arity))
				return Bool(true)
			}, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})
	})

	t.Run("term is not instantiated", func(t *testing.T) {
		t.Run("atom", func(t *testing.T) {
			env := Env{}
			term := NewVariable()
			ok, err := Functor(term, Atom("foo"), Integer(0), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)

			assert.Equal(t, Atom("foo"), env.Resolve(term))
		})

		t.Run("compound", func(t *testing.T) {
			env := Env{}
			term := NewVariable()
			ok, err := Functor(term, Atom("f"), Integer(2), func(env Env) Promise {
				c, ok := env.Resolve(term).(*Compound)
				assert.True(t, ok)
				assert.Equal(t, Atom("f"), c.Functor)
				assert.Len(t, c.Args, 2)
				assert.True(t, c.Args[0].(Variable).Anonymous())
				assert.True(t, c.Args[1].(Variable).Anonymous())
				return Bool(true)
			}, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})

		t.Run("name is not an atom", func(t *testing.T) {
			env := Env{}
			ok, err := Functor(NewVariable(), Integer(0), Integer(2), Success, &env).Force()
			assert.Equal(t, &Exception{
				Term: &Compound{
					Functor: "error",
					Args: []Term{
						&Compound{
							Functor: "type_error",
							Args: []Term{
								Atom("atom"),
								Integer(0),
							},
						},
						Atom("0 is not an atom."),
					},
				},
			}, err)
			assert.False(t, ok)
		})

		t.Run("arity is not an integer", func(t *testing.T) {
			env := Env{}
			ok, err := Functor(NewVariable(), Atom("f"), Float(2.0), Success, &env).Force()
			assert.Equal(t, &Exception{
				Term: &Compound{
					Functor: "error",
					Args: []Term{
						&Compound{
							Functor: "type_error",
							Args: []Term{
								Atom("integer"),
								Float(2.0),
							},
						},
						Atom("2 is not an integer."), // TODO: should it be 2.0?
					},
				},
			}, err)
			assert.False(t, ok)
		})

		t.Run("arity is negative", func(t *testing.T) {
			env := Env{}
			ok, err := Functor(NewVariable(), Atom("f"), Integer(-2), Success, &env).Force()
			assert.Equal(t, &Exception{
				Term: &Compound{
					Functor: "error",
					Args: []Term{
						&Compound{
							Functor: "domain_error",
							Args: []Term{
								Atom("not_less_than_zero"),
								Integer(-2),
							},
						},
						Atom("-2 is less than zero."),
					},
				},
			}, err)
			assert.False(t, ok)
		})
	})
}

func TestArg(t *testing.T) {
	t.Run("term is not a compound", func(t *testing.T) {
		env := Env{}
		ok, err := Arg(NewVariable(), Atom("foo"), NewVariable(), Success, &env).Force()
		assert.Equal(t, typeErrorCompound(Atom("foo")), err)
		assert.False(t, ok)
	})

	t.Run("nth is a variable", func(t *testing.T) {
		var (
			env = Env{}
			nth = NewVariable()
			c   int
		)
		ok, err := Arg(nth, &Compound{
			Functor: "f",
			Args:    []Term{Atom("a"), Atom("b"), Atom("a")},
		}, Atom("a"), func(env Env) Promise {
			switch c {
			case 0:
				assert.Equal(t, Integer(1), env.Resolve(nth))
			case 1:
				assert.Equal(t, Integer(3), env.Resolve(nth))
			default:
				assert.Fail(t, "unreachable")
			}
			c++
			return Bool(false)
		}, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("nth is an integer", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			env := Env{}
			ok, err := Arg(Integer(2), &Compound{
				Functor: "f",
				Args:    []Term{Atom("a"), Atom("b"), Atom("c")},
			}, Atom("b"), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})

		t.Run("ng", func(t *testing.T) {
			env := Env{}
			ok, err := Arg(Integer(4), &Compound{
				Functor: "f",
				Args:    []Term{Atom("a"), Atom("b"), Atom("c")},
			}, Atom("b"), Success, &env).Force()
			assert.NoError(t, err)
			assert.False(t, ok)
		})

		t.Run("negative", func(t *testing.T) {
			env := Env{}
			ok, err := Arg(Integer(-2), &Compound{
				Functor: "f",
				Args:    []Term{Atom("a"), Atom("b"), Atom("c")},
			}, Atom("b"), Success, &env).Force()
			assert.Equal(t, domainErrorNotLessThanZero(Integer(-2)), err)
			assert.False(t, ok)
		})
	})

	t.Run("nth is neither a variable nor an integer", func(t *testing.T) {
		env := Env{}
		ok, err := Arg(Atom("foo"), &Compound{
			Functor: "f",
			Args:    []Term{Atom("a"), Atom("b"), Atom("c")},
		}, Atom("b"), Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Atom("foo")), err)
		assert.False(t, ok)
	})
}

func TestUniv(t *testing.T) {
	t.Run("term is a variable", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			env := Env{}
			term := NewVariable()
			ok, err := Univ(term, List(Atom("f"), Atom("a"), Atom("b")), func(env Env) Promise {
				assert.Equal(t, &Compound{
					Functor: "f",
					Args:    []Term{Atom("a"), Atom("b")},
				}, env.Resolve(term))
				return Bool(true)
			}, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})

		t.Run("list is empty", func(t *testing.T) {
			env := Env{}
			term := NewVariable()
			ok, err := Univ(term, List(), Success, &env).Force()
			assert.Equal(t, domainErrorNotEmptyList(Atom("[]")), err)
			assert.False(t, ok)
		})

		t.Run("list is not a list", func(t *testing.T) {
			env := Env{}
			term := NewVariable()
			ok, err := Univ(term, Atom("list"), Success, &env).Force()
			assert.Equal(t, typeErrorList(Atom("list")), err)
			assert.False(t, ok)
		})

		t.Run("list's first element is not an atom", func(t *testing.T) {
			env := Env{}
			term := NewVariable()
			ok, err := Univ(term, List(Integer(0), Atom("a"), Atom("b")), Success, &env).Force()
			assert.Equal(t, typeErrorAtom(Integer(0)), err)
			assert.False(t, ok)
		})

		t.Run("list is not fully instantiated", func(t *testing.T) {
			env := Env{}
			term, rest := NewVariable(), Variable("Rest")
			ok, err := Univ(term, ListRest(rest, Atom("f"), Atom("a"), Atom("b")), Success, &env).Force()
			assert.Equal(t, instantiationError(ListRest(rest, Atom("a"), Atom("b"))), err)
			assert.False(t, ok)
		})
	})

	t.Run("term is a compound", func(t *testing.T) {
		env := Env{}
		ok, err := Univ(&Compound{
			Functor: "f",
			Args:    []Term{Atom("a"), Atom("b")},
		}, List(Atom("f"), Atom("a"), Atom("b")), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("term is neither a variable nor a compound", func(t *testing.T) {
		env := Env{}
		ok, err := Univ(Atom("foo"), List(Atom("foo")), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})
}

func TestCopyTerm(t *testing.T) {
	in := Variable("In")
	out := Variable("Out")
	env := Env{
		{
			Variable: in,
			Value:    Atom("a"),
		},
	}
	ok, err := CopyTerm(in, out, Success, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, Atom("a"), env.Resolve(out))
}

func TestVM_Op(t *testing.T) {
	t.Run("insert", func(t *testing.T) {
		env := Env{}
		vm := VM{
			operators: Operators{
				{
					Priority:  900,
					Specifier: "xfx",
					Name:      "+++",
				},
				{
					Priority:  1100,
					Specifier: "xfx",
					Name:      "+",
				},
			},
		}
		ok, err := vm.Op(Integer(1000), Atom("xfx"), Atom("++"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		assert.Equal(t, Operators{
			{
				Priority:  900,
				Specifier: "xfx",
				Name:      "+++",
			},
			{
				Priority:  1000,
				Specifier: "xfx",
				Name:      "++",
			},
			{
				Priority:  1100,
				Specifier: "xfx",
				Name:      "+",
			},
		}, vm.operators)
	})

	t.Run("remove", func(t *testing.T) {
		env := Env{}
		vm := VM{
			operators: Operators{
				{
					Priority:  900,
					Specifier: "xfx",
					Name:      "+++",
				},
				{
					Priority:  1000,
					Specifier: "xfx",
					Name:      "++",
				},
				{
					Priority:  1100,
					Specifier: "xfx",
					Name:      "+",
				},
			},
		}
		ok, err := vm.Op(Integer(0), Atom("xfx"), Atom("++"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		assert.Equal(t, Operators{
			{
				Priority:  900,
				Specifier: "xfx",
				Name:      "+++",
			},
			{
				Priority:  1100,
				Specifier: "xfx",
				Name:      "+",
			},
		}, vm.operators)
	})

	t.Run("priority is not an integer", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Op(Atom("foo"), Atom("xfx"), Atom("+"), Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Atom("foo")), err)
		assert.False(t, ok)
	})

	t.Run("priority is negative", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Op(Integer(-1), Atom("xfx"), Atom("+"), Success, &env).Force()
		assert.Equal(t, domainErrorOperatorPriority(Integer(-1)), err)
		assert.False(t, ok)
	})

	t.Run("priority is more than 1200", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Op(Integer(1201), Atom("xfx"), Atom("+"), Success, &env).Force()
		assert.Equal(t, domainErrorOperatorPriority(Integer(1201)), err)
		assert.False(t, ok)
	})

	t.Run("specifier is not an atom", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Op(Integer(1000), Integer(0), Atom("+"), Success, &env).Force()
		assert.Equal(t, typeErrorAtom(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("specifier is not a valid operator specifier", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Op(Integer(1000), Atom("foo"), Atom("+"), Success, &env).Force()
		assert.Equal(t, domainErrorOperatorSpecifier(Atom("foo")), err)
		assert.False(t, ok)
	})

	t.Run("operator is not an atom", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Op(Integer(1000), Atom("xfx"), Integer(0), Success, &env).Force()
		assert.Equal(t, typeErrorAtom(Integer(0)), err)
		assert.False(t, ok)
	})
}

func TestVM_CurrentOp(t *testing.T) {
	vm := VM{
		operators: Operators{
			{
				Priority:  900,
				Specifier: "xfx",
				Name:      "+++",
			},
			{
				Priority:  1000,
				Specifier: "xfx",
				Name:      "++",
			},
			{
				Priority:  1100,
				Specifier: "xfx",
				Name:      "+",
			},
		},
	}

	t.Run("single solution", func(t *testing.T) {
		env := Env{}
		ok, err := vm.CurrentOp(Integer(1100), Atom("xfx"), Atom("+"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("multiple solutions", func(t *testing.T) {
		var (
			env                           = Env{}
			priority, specifier, operator = Variable("Priority"), Variable("Specifier"), Variable("Operator")
			c                             int
		)
		ok, err := vm.CurrentOp(priority, specifier, operator, func(env Env) Promise {
			switch c {
			case 0:
				assert.Equal(t, Integer(900), env.Resolve(priority))
				assert.Equal(t, Atom("xfx"), env.Resolve(specifier))
				assert.Equal(t, Atom("+++"), env.Resolve(operator))
			case 1:
				assert.Equal(t, Integer(1000), env.Resolve(priority))
				assert.Equal(t, Atom("xfx"), env.Resolve(specifier))
				assert.Equal(t, Atom("++"), env.Resolve(operator))
			case 2:
				assert.Equal(t, Integer(1100), env.Resolve(priority))
				assert.Equal(t, Atom("xfx"), env.Resolve(specifier))
				assert.Equal(t, Atom("+"), env.Resolve(operator))
			default:
				assert.Fail(t, "unreachable")
			}
			c++
			return Bool(false)
		}, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("priority is not an operator priority", func(t *testing.T) {
		t.Run("priority is not an integer", func(t *testing.T) {
			env := Env{}
			ok, err := vm.CurrentOp(Atom("foo"), Atom("xfx"), Atom("+"), Success, &env).Force()
			assert.Equal(t, domainErrorOperatorPriority(Atom("foo")), err)
			assert.False(t, ok)
		})

		t.Run("priority is negative", func(t *testing.T) {
			env := Env{}
			ok, err := vm.CurrentOp(Integer(-1), Atom("xfx"), Atom("+"), Success, &env).Force()
			assert.Equal(t, domainErrorOperatorPriority(Integer(-1)), err)
			assert.False(t, ok)
		})
	})

	t.Run("specifier is not an operator specifier", func(t *testing.T) {
		t.Run("specifier is not an atom", func(t *testing.T) {
			env := Env{}
			ok, err := vm.CurrentOp(Integer(1100), Integer(0), Atom("+"), Success, &env).Force()
			assert.Equal(t, domainErrorOperatorSpecifier(Integer(0)), err)
			assert.False(t, ok)
		})

		t.Run("specifier is a non-specifier atom", func(t *testing.T) {
			env := Env{}
			ok, err := vm.CurrentOp(Integer(1100), Atom("foo"), Atom("+"), Success, &env).Force()
			assert.Equal(t, domainErrorOperatorSpecifier(Atom("foo")), err)
			assert.False(t, ok)
		})
	})

	t.Run("operator is not an atom", func(t *testing.T) {
		env := Env{}
		ok, err := vm.CurrentOp(Integer(1100), Atom("xfx"), Integer(0), Success, &env).Force()
		assert.Equal(t, typeErrorAtom(Integer(0)), err)
		assert.False(t, ok)
	})
}

func TestRepeat(t *testing.T) {
	env := Env{}
	c := 3
	ok, err := Repeat(func(env Env) Promise {
		c--
		return Bool(c == 0)
	}, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = Repeat(func(env Env) Promise {
		return Error(errors.New(""))
	}, &env).Force()
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestVM_BagOf(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		vm := VM{
			procedures: map[procedureIndicator]procedure{
				{name: "foo", arity: 3}: clauses{
					{xrTable: []Term{Atom("a"), Atom("b"), Atom("c")}, bytecode: bytecode{opConst, 0, opConst, 1, opConst, 2, opExit}},
					{xrTable: []Term{Atom("a"), Atom("b"), Atom("d")}, bytecode: bytecode{opConst, 0, opConst, 1, opConst, 2, opExit}},
					{xrTable: []Term{Atom("b"), Atom("c"), Atom("e")}, bytecode: bytecode{opConst, 0, opConst, 1, opConst, 2, opExit}},
					{xrTable: []Term{Atom("b"), Atom("c"), Atom("f")}, bytecode: bytecode{opConst, 0, opConst, 1, opConst, 2, opExit}},
					{xrTable: []Term{Atom("c"), Atom("c"), Atom("g")}, bytecode: bytecode{opConst, 0, opConst, 1, opConst, 2, opExit}},
				},
			},
		}

		t.Run("without qualifier", func(t *testing.T) {
			var (
				env         = Env{}
				count       int
				a, b, c, cs = Variable("A"), Variable("B"), Variable("C"), Variable("Cs")
			)
			ok, err := vm.BagOf(c, &Compound{
				Functor: "foo",
				Args:    []Term{a, b, c},
			}, cs, func(env Env) Promise {
				switch count {
				case 0:
					assert.Equal(t, Atom("a"), env.Resolve(a))
					assert.Equal(t, Atom("b"), env.Resolve(b))
					assert.Equal(t, c, env.Resolve(c))
					assert.True(t, List(Atom("c"), Atom("d")).Unify(cs, false, &env))
				case 1:
					assert.Equal(t, Atom("b"), env.Resolve(a))
					assert.Equal(t, Atom("c"), env.Resolve(b))
					assert.Equal(t, c, env.Resolve(c))
					assert.True(t, List(Atom("e"), Atom("f")).Unify(cs, false, &env))
				case 2:
					assert.Equal(t, Atom("c"), env.Resolve(a))
					assert.Equal(t, Atom("c"), env.Resolve(b))
					assert.Equal(t, c, env.Resolve(c))
					assert.True(t, List(Atom("g")).Unify(cs, false, &env))
				default:
					assert.Fail(t, "unreachable")
				}
				count++
				return Bool(false)
			}, &env).Force()
			assert.NoError(t, err)
			assert.False(t, ok)
		})

		t.Run("with qualifier", func(t *testing.T) {
			var (
				env         = Env{}
				count       int
				a, b, c, cs = Variable("A"), Variable("B"), Variable("C"), Variable("Cs")
			)
			ok, err := vm.BagOf(c, &Compound{
				Functor: "^",
				Args: []Term{a, &Compound{
					Functor: "foo",
					Args:    []Term{a, b, c},
				}},
			}, cs, func(env Env) Promise {
				switch count {
				case 0:
					assert.True(t, env.Resolve(a).(Variable).Anonymous())
					assert.Equal(t, Atom("b"), env.Resolve(b))
					assert.Equal(t, c, env.Resolve(c))
					assert.True(t, List(Atom("c"), Atom("d")).Unify(cs, false, &env))
				case 1:
					assert.True(t, env.Resolve(a).(Variable).Anonymous())
					assert.Equal(t, Atom("c"), env.Resolve(b))
					assert.Equal(t, c, env.Resolve(c))
					assert.True(t, List(Atom("e"), Atom("f"), Atom("g")).Unify(cs, false, &env))
				default:
					assert.Fail(t, "unreachable")
				}
				count++
				return Bool(false)
			}, &env).Force()
			assert.NoError(t, err)
			assert.False(t, ok)
		})

		t.Run("with multiple qualifiers", func(t *testing.T) {
			var (
				env         = Env{}
				count       int
				a, b, c, cs = Variable("A"), Variable("B"), Variable("C"), Variable("Cs")
			)
			ok, err := vm.BagOf(c, &Compound{
				Functor: "^",
				Args: []Term{
					&Compound{
						Functor: ",",
						Args:    []Term{a, b},
					},
					&Compound{
						Functor: "foo",
						Args:    []Term{a, b, c},
					},
				},
			}, cs, func(env Env) Promise {
				switch count {
				case 0:
					assert.Equal(t, a, env.Resolve(a))
					assert.Equal(t, b, env.Resolve(b))
					assert.Equal(t, c, env.Resolve(c))
					assert.True(t, List(Atom("c"), Atom("d"), Atom("e"), Atom("f"), Atom("g")).Unify(cs, false, &env))
				default:
					assert.Fail(t, "unreachable")
				}
				count++
				return Bool(false)
			}, &env).Force()
			assert.NoError(t, err)
			assert.False(t, ok)
		})
	})

	t.Run("goal is a variable", func(t *testing.T) {
		env := Env{}
		goal := Variable("Goal")

		var vm VM
		ok, err := vm.BagOf(NewVariable(), goal, NewVariable(), Success, &env).Force()
		assert.Equal(t, instantiationError(&goal), err)
		assert.False(t, ok)
	})

	t.Run("goal is neither a variable nor a callable term", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.BagOf(NewVariable(), Integer(0), NewVariable(), Success, &env).Force()
		assert.Equal(t, typeErrorCallable(Integer(0)), err)
		assert.False(t, ok)
	})
}

func TestSetOf(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		vm := VM{
			procedures: map[procedureIndicator]procedure{
				{name: "foo", arity: 3}: clauses{
					{xrTable: []Term{Atom("a"), Atom("b"), Atom("c")}, bytecode: bytecode{opConst, 0, opConst, 1, opConst, 2, opExit}},
					{xrTable: []Term{Atom("a"), Atom("b"), Atom("d")}, bytecode: bytecode{opConst, 0, opConst, 1, opConst, 2, opExit}},
					{xrTable: []Term{Atom("a"), Atom("b"), Atom("c")}, bytecode: bytecode{opConst, 0, opConst, 1, opConst, 2, opExit}},
					{xrTable: []Term{Atom("b"), Atom("c"), Atom("e")}, bytecode: bytecode{opConst, 0, opConst, 1, opConst, 2, opExit}},
					{xrTable: []Term{Atom("b"), Atom("c"), Atom("f")}, bytecode: bytecode{opConst, 0, opConst, 1, opConst, 2, opExit}},
					{xrTable: []Term{Atom("b"), Atom("c"), Atom("e")}, bytecode: bytecode{opConst, 0, opConst, 1, opConst, 2, opExit}},
					{xrTable: []Term{Atom("c"), Atom("c"), Atom("g")}, bytecode: bytecode{opConst, 0, opConst, 1, opConst, 2, opExit}},
					{xrTable: []Term{Atom("c"), Atom("c"), Atom("g")}, bytecode: bytecode{opConst, 0, opConst, 1, opConst, 2, opExit}},
				},
			},
		}

		t.Run("without qualifier", func(t *testing.T) {
			var (
				env         = Env{}
				count       int
				a, b, c, cs = Variable("A"), Variable("B"), Variable("C"), Variable("Cs")
			)
			ok, err := vm.SetOf(c, &Compound{
				Functor: "foo",
				Args:    []Term{a, b, c},
			}, cs, func(env Env) Promise {
				switch count {
				case 0:
					assert.Equal(t, Atom("a"), env.Resolve(a))
					assert.Equal(t, Atom("b"), env.Resolve(b))
					assert.Equal(t, c, env.Resolve(c))
					assert.Equal(t, List(Atom("c"), Atom("d")), env.Resolve(cs))
					assert.True(t, List(Atom("c"), Atom("d")).Unify(cs, false, &env))
				case 1:
					assert.Equal(t, Atom("b"), env.Resolve(a))
					assert.Equal(t, Atom("c"), env.Resolve(b))
					assert.Equal(t, c, env.Resolve(c))
					assert.True(t, List(Atom("e"), Atom("f")).Unify(cs, false, &env))
				case 2:
					assert.Equal(t, Atom("c"), env.Resolve(a))
					assert.Equal(t, Atom("c"), env.Resolve(b))
					assert.Equal(t, c, env.Resolve(c))
					assert.True(t, List(Atom("g")).Unify(cs, false, &env))
				default:
					assert.Fail(t, "unreachable")
				}
				count++
				return Bool(false)
			}, &env).Force()
			assert.NoError(t, err)
			assert.False(t, ok)
		})

		t.Run("with qualifier", func(t *testing.T) {
			var (
				env         = Env{}
				count       int
				a, b, c, cs = Variable("A"), Variable("B"), Variable("C"), Variable("Cs")
			)
			ok, err := vm.SetOf(c, &Compound{
				Functor: "^",
				Args: []Term{a, &Compound{
					Functor: "foo",
					Args:    []Term{a, b, c},
				}},
			}, cs, func(env Env) Promise {
				switch count {
				case 0:
					assert.True(t, env.Resolve(a).(Variable).Anonymous())
					assert.Equal(t, Atom("b"), env.Resolve(b))
					assert.Equal(t, c, env.Resolve(c))
					assert.True(t, List(Atom("c"), Atom("d")).Unify(cs, false, &env))
				case 1:
					assert.True(t, env.Resolve(a).(Variable).Anonymous())
					assert.Equal(t, Atom("c"), env.Resolve(b))
					assert.Equal(t, c, env.Resolve(c))
					assert.True(t, List(Atom("e"), Atom("f"), Atom("g")).Unify(cs, false, &env))
				default:
					assert.Fail(t, "unreachable")
				}
				count++
				return Bool(false)
			}, &env).Force()
			assert.NoError(t, err)
			assert.False(t, ok)
		})

		t.Run("with multiple qualifiers", func(t *testing.T) {
			var (
				env         = Env{}
				count       int
				a, b, c, cs = Variable("A"), Variable("B"), Variable("C"), Variable("Cs")
			)
			ok, err := vm.SetOf(c, &Compound{
				Functor: "^",
				Args: []Term{
					&Compound{
						Functor: ",",
						Args:    []Term{a, b},
					},
					&Compound{
						Functor: "foo",
						Args:    []Term{a, b, c},
					},
				},
			}, cs, func(env Env) Promise {
				switch count {
				case 0:
					assert.Equal(t, a, env.Resolve(a))
					assert.Equal(t, b, env.Resolve(b))
					assert.Equal(t, c, env.Resolve(c))
					assert.True(t, List(Atom("c"), Atom("d"), Atom("e"), Atom("f"), Atom("g")).Unify(cs, false, &env))
				default:
					assert.Fail(t, "unreachable")
				}
				count++
				return Bool(false)
			}, &env).Force()
			assert.NoError(t, err)
			assert.False(t, ok)
		})
	})

	t.Run("goal is a variable", func(t *testing.T) {
		env := Env{}
		goal := Variable("Goal")

		var vm VM
		ok, err := vm.SetOf(NewVariable(), goal, NewVariable(), Success, &env).Force()
		assert.Equal(t, instantiationError(goal), err)
		assert.False(t, ok)
	})

	t.Run("goal is neither a variable nor a callable term", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.SetOf(NewVariable(), Integer(0), NewVariable(), Success, &env).Force()
		assert.Equal(t, typeErrorCallable(Integer(0)), err)
		assert.False(t, ok)
	})
}

func TestCompare(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		env := Env{}

		x, y := Variable("X"), Variable("Y")
		ok, err := Compare(Atom("<"), x, y, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom("="), x, x, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom(">"), y, x, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		env = append(env, Binding{
			Variable: x,
			Value:    Atom("b"),
		}, Binding{
			Variable: y,
			Value:    Atom("a"),
		})
		ok, err = Compare(Atom(">"), x, y, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom("<"), NewVariable(), Integer(0), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom("<"), NewVariable(), Atom(""), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom("<"), NewVariable(), &Compound{}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom(">"), Integer(0), NewVariable(), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom("<"), Integer(0), Integer(1), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom("="), Integer(0), Integer(0), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom(">"), Integer(1), Integer(0), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom("<"), Integer(0), Atom(""), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom("<"), Integer(0), &Compound{}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom(">"), Atom(""), NewVariable(), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom(">"), Atom(""), Integer(0), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom("<"), Atom("a"), Atom("b"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom("="), Atom("a"), Atom("a"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom(">"), Atom("b"), Atom("a"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom("<"), Atom(""), &Compound{}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom(">"), &Compound{}, NewVariable(), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom(">"), &Compound{}, Integer(0), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom(">"), &Compound{}, Atom(""), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom("<"), &Compound{Functor: "a"}, &Compound{Functor: "b"}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom("="), &Compound{Functor: "a"}, &Compound{Functor: "a"}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom(">"), &Compound{Functor: "b"}, &Compound{Functor: "a"}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom(">"), &Compound{Functor: "f", Args: []Term{Atom("a")}}, &Compound{Functor: "f"}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom("="), &Compound{Functor: "f", Args: []Term{Atom("a")}}, &Compound{Functor: "f", Args: []Term{Atom("a")}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom("<"), &Compound{Functor: "f"}, &Compound{Functor: "f", Args: []Term{Atom("a")}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom(">"), &Compound{Functor: "f", Args: []Term{Atom("b")}}, &Compound{Functor: "f", Args: []Term{Atom("a")}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = Compare(Atom("<"), &Compound{Functor: "f", Args: []Term{Atom("a")}}, &Compound{Functor: "f", Args: []Term{Atom("b")}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("detect order", func(t *testing.T) {
		env := Env{}

		order := Variable("Order")
		ok, err := Compare(order, Atom("a"), Atom("b"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		assert.Equal(t, Atom("<"), env.Resolve(order))
	})

	t.Run("order is neither a variable nor an atom", func(t *testing.T) {
		env := Env{}
		ok, err := Compare(Integer(0), NewVariable(), NewVariable(), Success, &env).Force()
		assert.Equal(t, typeErrorAtom(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("order is an atom but not <, =, or >", func(t *testing.T) {
		env := Env{}
		ok, err := Compare(Atom("foo"), NewVariable(), NewVariable(), Success, &env).Force()
		assert.Equal(t, domainErrorOrder(Atom("foo")), err)
		assert.False(t, ok)
	})
}

func TestThrow(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		env := Env{}
		ok, err := Throw(Atom("a"), Success, &env).Force()
		assert.Equal(t, &Exception{Term: Atom("a")}, err)
		assert.False(t, ok)
	})

	t.Run("ball is a variable", func(t *testing.T) {
		env := Env{}
		ball := Variable("Ball")

		ok, err := Throw(ball, Success, &env).Force()
		assert.Equal(t, instantiationError(&ball), err)
		assert.False(t, ok)
	})
}

func TestVM_Catch(t *testing.T) {
	var vm VM
	vm.Register2("=", Unify)
	vm.Register1("throw", Throw)
	vm.Register0("true", func(k func(Env) Promise, env *Env) Promise {
		return k(*env)
	})
	vm.Register0("fail", func(_ func(Env) Promise, _ *Env) Promise {
		return Bool(false)
	})

	t.Run("match", func(t *testing.T) {
		env := Env{}
		v := NewVariable()
		ok, err := vm.Catch(&Compound{
			Functor: "throw",
			Args:    []Term{Atom("a")},
		}, v, &Compound{
			Functor: "=",
			Args:    []Term{v, Atom("a")},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("not match", func(t *testing.T) {
		env := Env{}
		ok, err := vm.Catch(&Compound{
			Functor: "throw",
			Args:    []Term{Atom("a")},
		}, Atom("b"), Atom("fail"), Success, &env).Force()
		assert.Equal(t, &Exception{Term: Atom("a")}, err)
		assert.False(t, ok)
	})

	t.Run("true", func(t *testing.T) {
		env := Env{}
		ok, err := vm.Catch(Atom("true"), Atom("b"), Atom("fail"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("false", func(t *testing.T) {
		env := Env{}
		ok, err := vm.Catch(Atom("fail"), Atom("b"), Atom("fail"), Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("non-exception error", func(t *testing.T) {
		env := Env{}
		ok, err := vm.Catch(Atom("true"), NewVariable(), Atom("true"), func(env Env) Promise {
			return Error(errors.New("failed"))
		}, &env).Force()
		assert.Error(t, err)
		assert.False(t, ok)
	})
}

func TestVM_CurrentPredicate(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		env := Env{}
		v := Variable("V")

		vm := VM{procedures: map[procedureIndicator]procedure{
			{name: "=", arity: 2}: nil,
		}}
		ok, err := vm.CurrentPredicate(v, func(env Env) Promise {
			assert.Equal(t, &Compound{
				Functor: "/",
				Args: []Term{
					Atom("="),
					Integer(2),
				},
			}, env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = vm.CurrentPredicate(v, Failure, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("pi is neither a variable nor a predicate indicator", func(t *testing.T) {
		t.Run("atom", func(t *testing.T) {
			env := Env{}
			var vm VM
			ok, err := vm.CurrentPredicate(Atom("foo"), Success, &env).Force()
			assert.Equal(t, typeErrorPredicateIndicator(Atom("foo")), err)
			assert.False(t, ok)
		})

		t.Run("compound", func(t *testing.T) {
			t.Run("non slash", func(t *testing.T) {
				env := Env{}
				var vm VM
				ok, err := vm.CurrentPredicate(&Compound{
					Functor: "f",
					Args:    []Term{Atom("a")},
				}, Success, &env).Force()
				assert.Equal(t, typeErrorPredicateIndicator(&Compound{
					Functor: "f",
					Args:    []Term{Atom("a")},
				}), err)
				assert.False(t, ok)
			})

			t.Run("slash but number", func(t *testing.T) {
				env := Env{}
				var vm VM
				ok, err := vm.CurrentPredicate(&Compound{
					Functor: "/",
					Args:    []Term{Integer(0), Integer(0)},
				}, Success, &env).Force()
				assert.Equal(t, typeErrorPredicateIndicator(&Compound{
					Functor: "/",
					Args:    []Term{Integer(0), Integer(0)},
				}), err)
				assert.False(t, ok)
			})

			t.Run("slash but path", func(t *testing.T) {
				env := Env{}
				var vm VM
				ok, err := vm.CurrentPredicate(&Compound{
					Functor: "/",
					Args:    []Term{Atom("foo"), Atom("bar")},
				}, Success, &env).Force()
				assert.Equal(t, typeErrorPredicateIndicator(&Compound{
					Functor: "/",
					Args:    []Term{Atom("foo"), Atom("bar")},
				}), err)
				assert.False(t, ok)
			})
		})
	})
}

func TestVM_Assertz(t *testing.T) {
	t.Run("append", func(t *testing.T) {
		env := Env{}
		var vm VM

		ok, err := vm.Assertz(&Compound{
			Functor: "foo",
			Args:    []Term{Atom("a")},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = vm.Assertz(&Compound{
			Functor: "foo",
			Args:    []Term{Atom("b")},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		assert.Equal(t, clauses{
			{
				pi: procedureIndicator{
					name:  "foo",
					arity: 1,
				},
				raw: &Compound{
					Functor: "foo",
					Args:    []Term{Atom("a")},
				},
				xrTable:  []Term{Atom("a")},
				bytecode: bytecode{opConst, 0, opExit},
			},
			{
				pi: procedureIndicator{
					name:  "foo",
					arity: 1,
				},
				raw: &Compound{
					Functor: "foo",
					Args:    []Term{Atom("b")},
				},
				xrTable:  []Term{Atom("b")},
				bytecode: bytecode{opConst, 0, opExit},
			},
		}, vm.procedures[procedureIndicator{
			name:  "foo",
			arity: 1,
		}])
	})

	t.Run("directive", func(t *testing.T) {
		env := Env{}

		var called bool
		vm := VM{
			procedures: map[procedureIndicator]procedure{
				{name: "directive", arity: 0}: predicate0(func(k func(Env) Promise, env *Env) Promise {
					called = true
					return k(*env)
				}),
			},
		}

		ok, err := vm.Assertz(&Compound{
			Functor: ":-",
			Args:    []Term{Atom("directive")},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		assert.True(t, called)
	})

	t.Run("clause is a variable", func(t *testing.T) {
		env := Env{}
		clause := Variable("Term")

		var vm VM
		ok, err := vm.Assertz(clause, Success, &env).Force()
		assert.Equal(t, instantiationError(&clause), err)
		assert.False(t, ok)
	})

	t.Run("clause is neither a variable, nor callable", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Assertz(Integer(0), Success, &env).Force()
		assert.Equal(t, typeErrorCallable(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("head is a variable", func(t *testing.T) {
		env := Env{}
		head := Variable("Head")

		var vm VM
		ok, err := vm.Assertz(&Compound{
			Functor: ":-",
			Args:    []Term{head, Atom("true")},
		}, Success, &env).Force()
		assert.Equal(t, instantiationError(&head), err)
		assert.False(t, ok)
	})

	t.Run("head is neither a variable, nor callable", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Assertz(&Compound{
			Functor: ":-",
			Args:    []Term{Integer(0), Atom("true")},
		}, Success, &env).Force()
		assert.Equal(t, typeErrorCallable(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("directive is a variable", func(t *testing.T) {
		env := Env{}
		directive := Variable("Directive")

		var vm VM
		ok, err := vm.Assertz(&Compound{
			Functor: ":-",
			Args:    []Term{directive},
		}, Success, &env).Force()
		assert.Equal(t, instantiationError(directive), err)
		assert.False(t, ok)
	})

	t.Run("directive is neither a variable, nor callable", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Assertz(&Compound{
			Functor: ":-",
			Args:    []Term{Integer(0)},
		}, Success, &env).Force()
		assert.Equal(t, typeErrorCallable(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("body contains a term which is not callable", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Assertz(&Compound{
			Functor: ":-",
			Args: []Term{Atom("foo"), &Compound{
				Functor: ",",
				Args:    []Term{Atom("true"), Integer(0)},
			}},
		}, Success, &env).Force()
		assert.Equal(t, typeErrorCallable(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("static", func(t *testing.T) {
		env := Env{}
		vm := VM{
			procedures: map[procedureIndicator]procedure{
				{name: "static", arity: 0}: predicate0(func(k func(Env) Promise, env *Env) Promise {
					return k(*env)
				}),
			},
		}

		ok, err := vm.Assertz(Atom("static"), Success, &env).Force()
		assert.Equal(t, permissionErrorModifyStaticProcedure(&Compound{
			Functor: "/",
			Args: []Term{
				Atom("static"),
				Integer(0),
			},
		}), err)
		assert.False(t, ok)
	})
}

func TestVM_Asserta(t *testing.T) {
	t.Run("prepend", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Asserta(&Compound{
			Functor: "foo",
			Args:    []Term{Atom("a")},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = vm.Asserta(&Compound{
			Functor: "foo",
			Args:    []Term{Atom("b")},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		assert.Equal(t, clauses{
			{
				pi: procedureIndicator{name: "foo", arity: 1},
				raw: &Compound{
					Functor: "foo",
					Args:    []Term{Atom("b")},
				},
				xrTable:  []Term{Atom("b")},
				bytecode: bytecode{opConst, 0, opExit},
			},
			{
				pi: procedureIndicator{name: "foo", arity: 1},
				raw: &Compound{
					Functor: "foo",
					Args:    []Term{Atom("a")},
				},
				xrTable:  []Term{Atom("a")},
				bytecode: bytecode{opConst, 0, opExit},
			},
		}, vm.procedures[procedureIndicator{name: "foo", arity: 1}])
	})

	t.Run("directive", func(t *testing.T) {
		env := Env{}
		var called bool
		vm := VM{
			procedures: map[procedureIndicator]procedure{
				{name: "directive", arity: 0}: predicate0(func(k func(Env) Promise, env *Env) Promise {
					called = true
					return k(*env)
				}),
			},
		}

		ok, err := vm.Asserta(&Compound{
			Functor: ":-",
			Args:    []Term{Atom("directive")},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		assert.True(t, called)
	})

	t.Run("clause is a variable", func(t *testing.T) {
		env := Env{}
		clause := Variable("Term")

		var vm VM
		ok, err := vm.Asserta(clause, Success, &env).Force()
		assert.Equal(t, instantiationError(&clause), err)
		assert.False(t, ok)
	})

	t.Run("clause is neither a variable, nor callable", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Asserta(Integer(0), Success, &env).Force()
		assert.Equal(t, typeErrorCallable(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("head is a variable", func(t *testing.T) {
		env := Env{}
		head := Variable("Head")

		var vm VM
		ok, err := vm.Asserta(&Compound{
			Functor: ":-",
			Args:    []Term{head, Atom("true")},
		}, Success, &env).Force()
		assert.Equal(t, instantiationError(head), err)
		assert.False(t, ok)
	})

	t.Run("head is neither a variable, nor callable", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Asserta(&Compound{
			Functor: ":-",
			Args:    []Term{Integer(0), Atom("true")},
		}, Success, &env).Force()
		assert.Equal(t, typeErrorCallable(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("directive is a variable", func(t *testing.T) {
		env := Env{}
		directive := Variable("Directive")

		var vm VM
		ok, err := vm.Asserta(&Compound{
			Functor: ":-",
			Args:    []Term{directive},
		}, Success, &env).Force()
		assert.Equal(t, instantiationError(directive), err)
		assert.False(t, ok)
	})

	t.Run("directive is neither a variable, nor callable", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Asserta(&Compound{
			Functor: ":-",
			Args:    []Term{Integer(0)},
		}, Success, &env).Force()
		assert.Equal(t, typeErrorCallable(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("body contains a term which is not callable", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Asserta(&Compound{
			Functor: ":-",
			Args: []Term{Atom("foo"), &Compound{
				Functor: ",",
				Args:    []Term{Atom("true"), Integer(0)},
			}},
		}, Success, &env).Force()
		assert.Equal(t, typeErrorCallable(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("static", func(t *testing.T) {
		env := Env{}
		vm := VM{
			procedures: map[procedureIndicator]procedure{
				{name: "static", arity: 0}: predicate0(func(k func(Env) Promise, env *Env) Promise {
					return k(*env)
				}),
			},
		}

		ok, err := vm.Asserta(Atom("static"), Success, &env).Force()
		assert.Equal(t, permissionErrorModifyStaticProcedure(&Compound{
			Functor: "/",
			Args: []Term{
				Atom("static"),
				Integer(0),
			},
		}), err)
		assert.False(t, ok)
	})
}

func TestVM_Retract(t *testing.T) {
	t.Run("retract the first one", func(t *testing.T) {
		env := Env{}
		vm := VM{
			procedures: map[procedureIndicator]procedure{
				{name: "foo", arity: 1}: clauses{
					{raw: &Compound{Functor: "foo", Args: []Term{Atom("a")}}},
					{raw: &Compound{Functor: "foo", Args: []Term{Atom("b")}}},
					{raw: &Compound{Functor: "foo", Args: []Term{Atom("c")}}},
				},
			},
		}

		ok, err := vm.Retract(&Compound{
			Functor: "foo",
			Args:    []Term{Variable("X")},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		assert.Equal(t, clauses{
			{raw: &Compound{Functor: "foo", Args: []Term{Atom("b")}}},
			{raw: &Compound{Functor: "foo", Args: []Term{Atom("c")}}},
		}, vm.procedures[procedureIndicator{name: "foo", arity: 1}])
	})

	t.Run("retract the specific one", func(t *testing.T) {
		env := Env{}
		vm := VM{
			procedures: map[procedureIndicator]procedure{
				{name: "foo", arity: 1}: clauses{
					{raw: &Compound{Functor: "foo", Args: []Term{Atom("a")}}},
					{raw: &Compound{Functor: "foo", Args: []Term{Atom("b")}}},
					{raw: &Compound{Functor: "foo", Args: []Term{Atom("c")}}},
				},
			},
		}

		ok, err := vm.Retract(&Compound{
			Functor: "foo",
			Args:    []Term{Atom("b")},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		assert.Equal(t, clauses{
			{raw: &Compound{Functor: "foo", Args: []Term{Atom("a")}}},
			{raw: &Compound{Functor: "foo", Args: []Term{Atom("c")}}},
		}, vm.procedures[procedureIndicator{name: "foo", arity: 1}])
	})

	t.Run("retract all", func(t *testing.T) {
		env := Env{}
		vm := VM{
			procedures: map[procedureIndicator]procedure{
				{name: "foo", arity: 1}: clauses{
					{raw: &Compound{Functor: "foo", Args: []Term{Atom("a")}}},
					{raw: &Compound{Functor: "foo", Args: []Term{Atom("b")}}},
					{raw: &Compound{Functor: "foo", Args: []Term{Atom("c")}}},
				},
			},
		}
		ok, err := vm.Retract(&Compound{
			Functor: "foo",
			Args:    []Term{Variable("X")},
		}, Failure, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)

		assert.Empty(t, vm.procedures[procedureIndicator{name: "foo", arity: 1}])
	})

	t.Run("variable", func(t *testing.T) {
		env := Env{}
		x := Variable("X")

		var vm VM
		ok, err := vm.Retract(x, Success, &env).Force()
		assert.Equal(t, instantiationError(x), err)
		assert.False(t, ok)
	})

	t.Run("not callable", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Retract(Integer(0), Success, &env).Force()
		assert.Equal(t, typeErrorCallable(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("no clause matches", func(t *testing.T) {
		env := Env{}
		var vm VM

		ok, err := vm.Retract(&Compound{
			Functor: "foo",
			Args:    []Term{Variable("X")},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("static", func(t *testing.T) {
		env := Env{}
		vm := VM{
			procedures: map[procedureIndicator]procedure{
				{name: "foo", arity: 0}: predicate0(nil),
			},
		}

		ok, err := vm.Retract(Atom("foo"), Success, &env).Force()
		assert.Equal(t, permissionErrorModifyStaticProcedure(&Compound{
			Functor: "/",
			Args:    []Term{Atom("foo"), Integer(0)},
		}), err)
		assert.False(t, ok)
	})

	t.Run("exception in continuation", func(t *testing.T) {
		env := Env{}
		vm := VM{
			procedures: map[procedureIndicator]procedure{
				{name: "foo", arity: 1}: clauses{
					{raw: &Compound{Functor: "foo", Args: []Term{Atom("a")}}},
				},
			},
		}

		ok, err := vm.Retract(&Compound{
			Functor: "foo",
			Args:    []Term{Variable("X")},
		}, func(_ Env) Promise {
			return Error(errors.New("failed"))
		}, &env).Force()
		assert.Error(t, err)
		assert.False(t, ok)

		// removed
		assert.Empty(t, vm.procedures[procedureIndicator{name: "foo", arity: 1}])
	})
}

func TestVM_Abolish(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		env := Env{}
		vm := VM{
			procedures: map[procedureIndicator]procedure{
				{name: "foo", arity: 1}: clauses{
					{raw: &Compound{Functor: "foo", Args: []Term{Atom("a")}}},
					{raw: &Compound{Functor: "foo", Args: []Term{Atom("b")}}},
					{raw: &Compound{Functor: "foo", Args: []Term{Atom("c")}}},
				},
			},
		}

		ok, err := vm.Abolish(&Compound{
			Functor: "/",
			Args:    []Term{Atom("foo"), Integer(1)},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		_, ok = vm.procedures[procedureIndicator{name: "foo", arity: 1}]
		assert.False(t, ok)
	})

	t.Run("pi is a variable", func(t *testing.T) {
		env := Env{}
		pi := Variable("PI")

		var vm VM
		ok, err := vm.Abolish(pi, Success, &env).Force()
		assert.Equal(t, instantiationError(&pi), err)
		assert.False(t, ok)
	})

	t.Run("pi is a term Name/Arity and either Name or Arity is a variable", func(t *testing.T) {
		t.Run("Name is a variable", func(t *testing.T) {
			env := Env{}
			name := Variable("Name")

			var vm VM
			ok, err := vm.Abolish(&Compound{
				Functor: "/",
				Args:    []Term{name, Integer(2)},
			}, Success, &env).Force()
			assert.Equal(t, instantiationError(name), err)
			assert.False(t, ok)
		})

		t.Run("Arity is a variable", func(t *testing.T) {
			env := Env{}
			arity := Variable("Arity")

			var vm VM
			ok, err := vm.Abolish(&Compound{
				Functor: "/",
				Args:    []Term{Atom("foo"), arity},
			}, Success, &env).Force()
			assert.Equal(t, instantiationError(arity), err)
			assert.False(t, ok)
		})
	})

	t.Run("pi is neither a variable nor a predicate indicator", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Abolish(Integer(0), Success, &env).Force()
		assert.Equal(t, typeErrorPredicateIndicator(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("pi is a term Name/Arity and Name is neither a variable nor an atom", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Abolish(&Compound{
			Functor: "/",
			Args:    []Term{Integer(0), Integer(2)},
		}, Success, &env).Force()
		assert.Equal(t, typeErrorAtom(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("pi is a term Name/Arity and Arity is neither a variable nor an integer", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Abolish(&Compound{
			Functor: "/",
			Args:    []Term{Atom("foo"), Atom("bar")},
		}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Atom("bar")), err)
		assert.False(t, ok)
	})

	t.Run("pi is a term Name/Arity and Arity is an integer less than zero", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Abolish(&Compound{
			Functor: "/",
			Args:    []Term{Atom("foo"), Integer(-2)},
		}, Success, &env).Force()
		assert.Equal(t, domainErrorNotLessThanZero(Integer(-2)), err)
		assert.False(t, ok)
	})

	t.Run("The predicate indicator pi is that of a static procedure", func(t *testing.T) {
		env := Env{}
		vm := VM{
			procedures: map[procedureIndicator]procedure{
				{name: "foo", arity: 0}: predicate0(nil),
			},
		}
		ok, err := vm.Abolish(&Compound{
			Functor: "/",
			Args:    []Term{Atom("foo"), Integer(0)},
		}, Success, &env).Force()
		assert.Equal(t, permissionErrorModifyStaticProcedure(&Compound{
			Functor: "/",
			Args:    []Term{Atom("foo"), Integer(0)},
		}), err)
		assert.False(t, ok)
	})
}

func TestVM_CurrentInput(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		env := Env{}
		var s Stream
		vm := VM{
			input: &s,
		}

		ok, err := vm.CurrentInput(&s, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("stream is neither a variable nor a stream", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.CurrentInput(Integer(0), Success, &env).Force()
		assert.Equal(t, domainErrorStream(Integer(0)), err)
		assert.False(t, ok)
	})
}

func TestVM_CurrentOutput(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		env := Env{}
		var s Stream
		vm := VM{
			output: &s,
		}

		ok, err := vm.CurrentOutput(&s, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("stream is neither a variable nor a stream", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.CurrentOutput(Integer(0), Success, &env).Force()
		assert.Equal(t, domainErrorStream(Integer(0)), err)
		assert.False(t, ok)
	})
}

func TestVM_SetInput(t *testing.T) {
	t.Run("stream", func(t *testing.T) {
		v := Variable("Stream")
		s := Stream{source: os.Stdin}
		env := Env{
			{
				Variable: v,
				Value:    &s,
			},
		}
		var vm VM
		ok, err := vm.SetInput(v, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, &s, vm.input)
	})

	t.Run("alias", func(t *testing.T) {
		v := Variable("Stream")
		s := Stream{source: os.Stdin}
		env := Env{
			{
				Variable: v,
				Value:    &s,
			},
		}
		vm := VM{
			streams: map[Term]*Stream{
				Atom("x"): &s,
			},
		}
		ok, err := vm.SetInput(v, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, &s, vm.input)
	})

	t.Run("streamOrAlias is a variable", func(t *testing.T) {
		env := Env{}
		streamOrAlias := Variable("Stream")

		var vm VM
		ok, err := vm.SetInput(streamOrAlias, Success, &env).Force()
		assert.Equal(t, instantiationError(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is neither a variable, nor a stream term or alias", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.SetInput(Integer(0), Success, &env).Force()
		assert.Equal(t, domainErrorStreamOrAlias(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is not associated with an open stream", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.SetInput(Atom("x"), Success, &env).Force()
		assert.Equal(t, existenceErrorStream(Atom("x")), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is an output stream", func(t *testing.T) {
		v := Variable("Stream")
		env := Env{
			{
				Variable: v,
				Value:    &Stream{sink: os.Stdout},
			},
		}
		var vm VM
		ok, err := vm.SetInput(v, Success, &env).Force()
		assert.Equal(t, permissionErrorInputStream(v), err)
		assert.False(t, ok)
	})
}

func TestVM_SetOutput(t *testing.T) {
	t.Run("stream", func(t *testing.T) {
		v := Variable("Stream")
		s := Stream{sink: os.Stdout}
		env := Env{
			{
				Variable: v,
				Value:    &s,
			},
		}
		var vm VM
		ok, err := vm.SetOutput(v, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, &s, vm.output)
	})

	t.Run("alias", func(t *testing.T) {
		env := Env{}
		s := Stream{sink: os.Stdout}
		vm := VM{
			streams: map[Term]*Stream{
				Atom("x"): &s,
			},
		}
		ok, err := vm.SetOutput(Atom("x"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, &s, vm.output)
	})

	t.Run("streamOrAlias is a variable", func(t *testing.T) {
		env := Env{}
		streamOrAlias := Variable("Stream")

		var vm VM
		ok, err := vm.SetOutput(streamOrAlias, Success, &env).Force()
		assert.Equal(t, instantiationError(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is neither a variable, nor a stream term or alias", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.SetOutput(Integer(0), Success, &env).Force()
		assert.Equal(t, domainErrorStreamOrAlias(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is not associated with an open stream", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.SetOutput(Atom("x"), Success, &env).Force()
		assert.Equal(t, existenceErrorStream(Atom("x")), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is an input stream", func(t *testing.T) {
		s := Variable("Stream")
		env := Env{
			{
				Variable: s,
				Value:    &Stream{source: os.Stdin},
			},
		}

		var vm VM
		ok, err := vm.SetOutput(s, Success, &env).Force()
		assert.Equal(t, permissionErrorOutputStream(s), err)
		assert.False(t, ok)
	})
}

func TestVM_Open(t *testing.T) {
	var vm VM

	t.Run("read", func(t *testing.T) {
		f, err := ioutil.TempFile("", "open_test_read")
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, os.Remove(f.Name()))
		}()

		_, err = fmt.Fprintf(f, "test\n")
		assert.NoError(t, err)

		assert.NoError(t, f.Close())

		env := Env{}
		v := Variable("Stream")

		ok, err := vm.Open(Atom(f.Name()), Atom("read"), v, List(&Compound{
			Functor: "alias",
			Args:    []Term{Atom("input")},
		}), func(env Env) Promise {
			ref, ok := env.Lookup(v)
			assert.True(t, ok)
			s, ok := ref.(*Stream)
			assert.True(t, ok)

			assert.Equal(t, vm.streams[Atom("input")], s)

			b, err := ioutil.ReadAll(s.source)
			assert.NoError(t, err)
			assert.Equal(t, "test\n", string(b))

			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("write", func(t *testing.T) {
		n := filepath.Join(os.TempDir(), "open_test_write")
		defer func() {
			assert.NoError(t, os.Remove(n))
		}()

		env := Env{}
		v := Variable("Stream")

		ok, err := vm.Open(Atom(n), Atom("write"), v, List(&Compound{
			Functor: "alias",
			Args:    []Term{Atom("output")},
		}), func(env Env) Promise {
			ref, ok := env.Lookup(v)
			assert.True(t, ok)
			s, ok := ref.(*Stream)
			assert.True(t, ok)

			assert.Equal(t, vm.streams[Atom("output")], s)

			_, err := fmt.Fprintf(s.sink, "test\n")
			assert.NoError(t, err)

			f, err := os.Open(n)
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, f.Close())
			}()

			b, err := ioutil.ReadAll(f)
			assert.NoError(t, err)
			assert.Equal(t, "test\n", string(b))

			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("append", func(t *testing.T) {
		f, err := ioutil.TempFile("", "open_test_append")
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, os.Remove(f.Name()))
		}()

		_, err = fmt.Fprintf(f, "test\n")
		assert.NoError(t, err)

		assert.NoError(t, f.Close())

		env := Env{}
		v := Variable("Stream")

		ok, err := vm.Open(Atom(f.Name()), Atom("append"), v, List(&Compound{
			Functor: "alias",
			Args:    []Term{Atom("append")},
		}), func(env Env) Promise {
			ref, ok := env.Lookup(v)
			assert.True(t, ok)
			s, ok := ref.(*Stream)
			assert.True(t, ok)

			assert.Equal(t, vm.streams[Atom("append")], s)

			_, err = fmt.Fprintf(s.sink, "test\n")
			assert.NoError(t, err)

			f, err = os.Open(f.Name())
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, f.Close())
			}()

			b, err := ioutil.ReadAll(f)
			assert.NoError(t, err)
			assert.Equal(t, "test\ntest\n", string(b))

			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("sourceSink is a variable", func(t *testing.T) {
		env := Env{}
		sourceSink := Variable("Source_Sink")

		var vm VM
		ok, err := vm.Open(sourceSink, Atom("read"), Variable("Stream"), List(), Success, &env).Force()
		assert.Equal(t, instantiationError(sourceSink), err)
		assert.False(t, ok)
	})

	t.Run("mode is a variable", func(t *testing.T) {
		env := Env{}
		mode := Variable("Mode")

		var vm VM
		ok, err := vm.Open(Atom("/dev/null"), mode, Variable("Stream"), List(), Success, &env).Force()
		assert.Equal(t, instantiationError(mode), err)
		assert.False(t, ok)
	})

	t.Run("options is a partial list or a list with an element E which is a variable", func(t *testing.T) {
		t.Run("partial list", func(t *testing.T) {
			env := Env{}
			options := ListRest(Variable("Rest"),
				&Compound{Functor: "type", Args: []Term{Atom("text")}},
				&Compound{Functor: "alias", Args: []Term{Atom("foo")}},
			)

			var vm VM
			ok, err := vm.Open(Atom("/dev/null"), Atom("read"), Variable("Stream"), options, Success, &env).Force()
			assert.Equal(t, instantiationError(options), err)
			assert.False(t, ok)
		})

		t.Run("variable element", func(t *testing.T) {
			env := Env{}
			option := Variable("Option")

			var vm VM
			ok, err := vm.Open(Atom("/dev/null"), Atom("read"), Variable("Stream"), List(
				option,
				&Compound{Functor: "type", Args: []Term{Atom("text")}},
				&Compound{Functor: "alias", Args: []Term{Atom("foo")}},
			), Success, &env).Force()
			assert.Equal(t, instantiationError(option), err)
			assert.False(t, ok)
		})
	})

	t.Run("mode is neither a variable nor an atom", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Open(Atom("/dev/null"), Integer(0), Variable("Stream"), List(), Success, &env).Force()
		assert.Equal(t, typeErrorAtom(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("options is neither a partial list nor a list", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Open(Atom("/dev/null"), Atom("read"), Variable("Stream"), Atom("list"), Success, &env).Force()
		assert.Equal(t, typeErrorList(Atom("list")), err)
		assert.False(t, ok)
	})

	t.Run("stream is not a variable", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Open(Atom("/dev/null"), Atom("read"), Atom("stream"), List(), Success, &env).Force()
		assert.Equal(t, typeErrorVariable(Atom("stream")), err)
		assert.False(t, ok)
	})

	t.Run("sourceSink is neither a variable nor a source/sink", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Open(Integer(0), Atom("read"), Variable("Stream"), List(), Success, &env).Force()
		assert.Equal(t, domainErrorSourceSink(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("mode is an atom but not an input/output mode", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Open(Atom("/dev/null"), Atom("foo"), Variable("Stream"), List(), Success, &env).Force()
		assert.Equal(t, domainErrorIOMode(Atom("foo")), err)
		assert.False(t, ok)
	})

	t.Run("an element E of the options list is neither a variable nor a stream-option", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Open(Atom("/dev/null"), Atom("read"), Variable("Stream"), List(Atom("foo")), Success, &env).Force()
		assert.Equal(t, domainErrorStreamOption(Atom("foo")), err)
		assert.False(t, ok)
	})

	t.Run("the source/sink specified by sourceSink does not exist", func(t *testing.T) {
		env := Env{}
		f, err := ioutil.TempFile("", "open_test_existence")
		assert.NoError(t, err)
		assert.NoError(t, os.Remove(f.Name()))

		var vm VM
		ok, err := vm.Open(Atom(f.Name()), Atom("read"), Variable("Stream"), List(), Success, &env).Force()
		assert.Equal(t, existenceErrorSourceSink(Atom(f.Name())), err)
		assert.False(t, ok)
	})

	t.Run("the source/sink specified by sourceSink cannot be opened", func(t *testing.T) {
		env := Env{}
		f, err := ioutil.TempFile("", "open_test_permission")
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, os.Remove(f.Name()))
		}()

		assert.NoError(t, f.Chmod(0200))

		var vm VM
		ok, err := vm.Open(Atom(f.Name()), Atom("read"), Variable("Stream"), List(), Success, &env).Force()
		assert.Equal(t, permissionError(Atom("open"), Atom("source_sink"), Atom(f.Name()), Atom(fmt.Sprintf("'%s' cannot be opened.", f.Name()))), err)
		assert.False(t, ok)
	})

	t.Run("an element E of the options list is alias and A is already associated with an open stream", func(t *testing.T) {
		env := Env{}
		f, err := ioutil.TempFile("", "open_test_dup_alias")
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, os.Remove(f.Name()))
		}()

		vm := VM{
			streams: map[Term]*Stream{
				Atom("foo"): nil,
			},
		}
		ok, err := vm.Open(Atom(f.Name()), Atom("read"), Variable("Stream"), List(&Compound{
			Functor: "alias",
			Args:    []Term{Atom("foo")},
		}), Success, &env).Force()
		assert.Equal(t, permissionError(Atom("open"), Atom("source_sink"), &Compound{
			Functor: "alias",
			Args:    []Term{Atom("foo")},
		}, Atom("foo is already defined as an alias.")), err)
		assert.False(t, ok)
	})

	t.Run("an element E of the options list is reposition(true) and it is not possible to reposition", func(t *testing.T) {
		// TODO:
	})
}

func TestVM_Close(t *testing.T) {
	t.Run("without options", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			env := Env{}

			var m mockCloser
			m.On("Close").Return(nil).Once()
			defer m.AssertExpectations(t)

			var vm VM
			ok, err := vm.Close(&Stream{closer: &m}, List(), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})

		t.Run("ng", func(t *testing.T) {
			env := Env{}

			var m mockCloser
			m.On("Close").Return(errors.New("")).Once()
			defer m.AssertExpectations(t)

			var vm VM
			_, err := vm.Close(&Stream{closer: &m}, List(), Success, &env).Force()
			assert.Error(t, err)
		})
	})

	t.Run("force false", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			env := Env{}

			var m mockCloser
			m.On("Close").Return(nil).Once()
			defer m.AssertExpectations(t)

			var vm VM
			ok, err := vm.Close(&Stream{closer: &m}, List(&Compound{
				Functor: "force",
				Args:    []Term{Atom("false")},
			}), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})

		t.Run("ng", func(t *testing.T) {
			env := Env{}

			var m mockCloser
			m.On("Close").Return(errors.New("something happened")).Once()
			defer m.AssertExpectations(t)

			s := Stream{closer: &m}

			var vm VM
			ok, err := vm.Close(&s, List(&Compound{
				Functor: "force",
				Args:    []Term{Atom("false")},
			}), Success, &env).Force()
			assert.Equal(t, resourceError(&s, Atom("something happened")), err)
			assert.False(t, ok)
		})
	})

	t.Run("force true", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			env := Env{}

			var m mockCloser
			m.On("Close").Return(nil).Once()
			defer m.AssertExpectations(t)

			var vm VM
			ok, err := vm.Close(&Stream{closer: &m}, List(&Compound{
				Functor: "force",
				Args:    []Term{Atom("true")},
			}), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})

		t.Run("ng", func(t *testing.T) {
			env := Env{}

			var m mockCloser
			m.On("Close").Return(errors.New("")).Once()
			defer m.AssertExpectations(t)

			var vm VM
			ok, err := vm.Close(&Stream{closer: &m}, List(&Compound{
				Functor: "force",
				Args:    []Term{Atom("true")},
			}), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})
	})

	t.Run("valid stream alias", func(t *testing.T) {
		env := Env{}

		var m mockCloser
		m.On("Close").Return(nil).Once()
		defer m.AssertExpectations(t)

		vm := VM{
			streams: map[Term]*Stream{
				Atom("foo"): {closer: &m},
			},
		}
		ok, err := vm.Close(Atom("foo"), List(), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("streamOrAlias ia a variable", func(t *testing.T) {
		env := Env{}
		streamOrAlias := Variable("Stream")

		var vm VM
		ok, err := vm.Close(streamOrAlias, List(), Success, &env).Force()
		assert.Equal(t, instantiationError(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("options is a partial list or a list with an element E which is a variable", func(t *testing.T) {
		t.Run("partial list", func(t *testing.T) {
			env := Env{}
			options := ListRest(Variable("Rest"),
				&Compound{Functor: "force", Args: []Term{Atom("true")}},
			)

			var vm VM
			ok, err := vm.Close(&Stream{}, options, Success, &env).Force()
			assert.Equal(t, instantiationError(options), err)
			assert.False(t, ok)
		})

		t.Run("variable element", func(t *testing.T) {
			env := Env{}
			option := Variable("Option")

			var vm VM
			ok, err := vm.Close(&Stream{}, List(option, &Compound{Functor: "force", Args: []Term{Atom("true")}}), Success, &env).Force()
			assert.Equal(t, instantiationError(option), err)
			assert.False(t, ok)
		})
	})

	t.Run("options is neither a partial list nor a list", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Close(&Stream{}, Atom("foo"), Success, &env).Force()
		assert.Equal(t, typeErrorList(Atom("foo")), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is neither a variable nor a stream-term or alias", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Close(Integer(0), List(), Success, &env).Force()
		assert.Equal(t, domainErrorStreamOrAlias(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("an element E of the Options list is neither a variable nor a stream-option", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Close(&Stream{}, List(Atom("foo")), Success, &env).Force()
		assert.Equal(t, domainErrorStreamOption(Atom("foo")), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is not associated with an open stream", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Close(Atom("foo"), List(), Success, &env).Force()
		assert.Equal(t, existenceErrorStream(Atom("foo")), err)
		assert.False(t, ok)
	})
}

type mockReader struct {
	mock.Mock
}

func (m *mockReader) Read(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

type mockWriter struct {
	mock.Mock
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

type mockCloser struct {
	mock.Mock
}

func (m *mockCloser) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestVM_FlushOutput(t *testing.T) {
	t.Run("non flusher", func(t *testing.T) {
		env := Env{}

		var m mockWriter
		defer m.AssertExpectations(t)

		var vm VM
		ok, err := vm.FlushOutput(&Stream{sink: &m}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("flusher", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			env := Env{}

			var m struct {
				mockWriter
				mockFlusher
			}
			m.mockFlusher.On("Flush").Return(nil).Once()
			defer m.mockWriter.AssertExpectations(t)
			defer m.mockFlusher.AssertExpectations(t)

			var vm VM
			ok, err := vm.FlushOutput(&Stream{sink: &m}, Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})

		t.Run("ng", func(t *testing.T) {
			env := Env{}

			var m struct {
				mockWriter
				mockFlusher
			}
			m.mockFlusher.On("Flush").Return(errors.New("")).Once()
			defer m.mockWriter.AssertExpectations(t)
			defer m.mockFlusher.AssertExpectations(t)

			var vm VM
			_, err := vm.FlushOutput(&Stream{sink: &m}, Success, &env).Force()
			assert.Error(t, err)
		})
	})

	t.Run("valid stream alias", func(t *testing.T) {
		env := Env{}

		var m mockWriter
		defer m.AssertExpectations(t)

		vm := VM{
			streams: map[Term]*Stream{
				Atom("foo"): {sink: &m},
			},
		}
		ok, err := vm.FlushOutput(Atom("foo"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("streamOrAlias is a variable", func(t *testing.T) {
		env := Env{}
		streamOrAlias := Variable("Stream")

		var vm VM
		ok, err := vm.FlushOutput(streamOrAlias, Success, &env).Force()
		assert.Equal(t, instantiationError(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is neither a variable nor a stream-term or alias", func(t *testing.T) {
		env := Env{}

		var vm VM
		ok, err := vm.FlushOutput(Integer(0), Success, &env).Force()
		assert.Equal(t, domainErrorStreamOrAlias(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is not associated with an open stream", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.FlushOutput(Atom("foo"), Success, &env).Force()
		assert.Equal(t, existenceErrorStream(Atom("foo")), err)
		assert.False(t, ok)
	})

	t.Run("SorA is an input stream", func(t *testing.T) {
		env := Env{}
		s := Stream{source: &mockReader{}}

		var vm VM
		ok, err := vm.FlushOutput(&s, Success, &env).Force()
		assert.Equal(t, permissionErrorOutputStream(&s), err)
		assert.False(t, ok)
	})
}

type mockFlusher struct {
	mock.Mock
}

func (m *mockFlusher) Flush() error {
	args := m.Called()
	return args.Error(0)
}

func TestVM_WriteTerm(t *testing.T) {
	var w mockWriter
	defer w.AssertExpectations(t)

	s := Stream{sink: &w}

	ops := Operators{
		{Priority: 500, Specifier: "yfx", Name: "+"},
		{Priority: 200, Specifier: "fy", Name: "-"},
	}

	vm := VM{
		operators: ops,
		streams: map[Term]*Stream{
			Atom("foo"): &s,
		},
	}

	t.Run("without options", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			env := Env{}

			var m mockTerm
			m.On("WriteTerm", s.sink, WriteTermOptions{Ops: ops}, env).Return(nil).Once()
			defer m.AssertExpectations(t)

			ok, err := vm.WriteTerm(&s, &m, List(), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})

		t.Run("ng", func(t *testing.T) {
			env := Env{}

			var m mockTerm
			m.On("WriteTerm", s.sink, WriteTermOptions{Ops: ops}, env).Return(errors.New("")).Once()
			defer m.AssertExpectations(t)

			_, err := vm.WriteTerm(&s, &m, List(), Success, &env).Force()
			assert.Error(t, err)
		})
	})

	t.Run("quoted", func(t *testing.T) {
		t.Run("false", func(t *testing.T) {
			env := Env{}

			var m mockTerm
			m.On("WriteTerm", s.sink, WriteTermOptions{Quoted: false, Ops: ops}, env).Return(nil).Once()
			defer m.AssertExpectations(t)

			ok, err := vm.WriteTerm(&s, &m, List(&Compound{
				Functor: "quoted",
				Args:    []Term{Atom("false")},
			}), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})

		t.Run("true", func(t *testing.T) {
			env := Env{}

			var m mockTerm
			m.On("WriteTerm", s.sink, WriteTermOptions{Quoted: true, Ops: ops}, env).Return(nil).Once()
			defer m.AssertExpectations(t)

			ok, err := vm.WriteTerm(&s, &m, List(&Compound{
				Functor: "quoted",
				Args:    []Term{Atom("true")},
			}), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})
	})

	t.Run("ignore_ops", func(t *testing.T) {
		t.Run("false", func(t *testing.T) {
			env := Env{}

			var m mockTerm
			m.On("WriteTerm", s.sink, WriteTermOptions{Ops: ops}, env).Return(nil).Once()
			defer m.AssertExpectations(t)

			ok, err := vm.WriteTerm(&s, &m, List(&Compound{
				Functor: "ignore_ops",
				Args:    []Term{Atom("false")},
			}), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})

		t.Run("true", func(t *testing.T) {
			env := Env{}

			var m mockTerm
			m.On("WriteTerm", s.sink, WriteTermOptions{Ops: nil}, env).Return(nil).Once()
			defer m.AssertExpectations(t)

			ok, err := vm.WriteTerm(&s, &m, List(&Compound{
				Functor: "ignore_ops",
				Args:    []Term{Atom("true")},
			}), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})
	})

	t.Run("numbervars", func(t *testing.T) {
		t.Run("false", func(t *testing.T) {
			env := Env{}

			var m mockTerm
			m.On("WriteTerm", s.sink, WriteTermOptions{Ops: ops, NumberVars: false}, env).Return(nil).Once()
			defer m.AssertExpectations(t)

			ok, err := vm.WriteTerm(&s, &m, List(&Compound{
				Functor: "numbervars",
				Args:    []Term{Atom("false")},
			}), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})

		t.Run("true", func(t *testing.T) {
			env := Env{}

			var m mockTerm
			m.On("WriteTerm", s.sink, WriteTermOptions{Ops: ops, NumberVars: true}, env).Return(nil).Once()
			defer m.AssertExpectations(t)

			ok, err := vm.WriteTerm(&s, &m, List(&Compound{
				Functor: "numbervars",
				Args:    []Term{Atom("true")},
			}), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})
	})

	t.Run("streamOrAlias is a variable", func(t *testing.T) {
		env := Env{}
		streamOrAlias := Variable("Stream")

		var vm VM
		ok, err := vm.WriteTerm(streamOrAlias, Atom("foo"), List(), Success, &env).Force()
		assert.Equal(t, instantiationError(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("options is a partial list or a list with an element which is a variable", func(t *testing.T) {
		t.Run("partial list", func(t *testing.T) {
			env := Env{}
			options := ListRest(Variable("Rest"),
				&Compound{Functor: "quoted", Args: []Term{Atom("true")}},
			)

			var vm VM
			ok, err := vm.WriteTerm(&Stream{sink: &mockWriter{}}, Atom("foo"), options, Success, &env).Force()
			assert.Equal(t, instantiationError(options), err)
			assert.False(t, ok)
		})

		t.Run("variable element", func(t *testing.T) {
			env := Env{}
			option := Variable("Option")

			var vm VM
			ok, err := vm.WriteTerm(&Stream{sink: &mockWriter{}}, Atom("foo"), List(option, &Compound{Functor: "quoted", Args: []Term{Atom("true")}}), Success, &env).Force()
			assert.Equal(t, instantiationError(option), err)
			assert.False(t, ok)
		})
	})

	t.Run("streamOrAlias is neither a variable nor a stream term or alias", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.WriteTerm(Integer(0), Atom("foo"), List(), Success, &env).Force()
		assert.Equal(t, domainErrorStreamOrAlias(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("options is neither a partial list nor a list", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.WriteTerm(&Stream{sink: &mockWriter{}}, Atom("foo"), Atom("options"), Success, &env).Force()
		assert.Equal(t, typeErrorList(Atom("options")), err)
		assert.False(t, ok)
	})

	t.Run("an element E of the Options list is neither a variable nor a valid write-option", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.WriteTerm(&Stream{sink: &mockWriter{}}, Atom("foo"), List(&Compound{
			Functor: "unknown",
			Args:    []Term{Atom("option")},
		}), Success, &env).Force()
		assert.Equal(t, domainErrorWriteOption(&Compound{
			Functor: "unknown",
			Args:    []Term{Atom("option")},
		}), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is not associated with an open stream", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.WriteTerm(Atom("stream"), Atom("foo"), List(), Success, &env).Force()
		assert.Equal(t, existenceErrorStream(Atom("stream")), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is an input stream", func(t *testing.T) {
		env := Env{}
		s := Stream{source: &mockReader{}}

		var vm VM
		ok, err := vm.WriteTerm(&s, Atom("foo"), List(), Success, &env).Force()
		assert.Equal(t, permissionErrorOutputStream(&s), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is associated with a binary stream", func(t *testing.T) {
		env := Env{}
		s := Stream{sink: &mockWriter{}, streamType: streamTypeBinary}

		var vm VM
		ok, err := vm.WriteTerm(&s, Atom("foo"), List(), Success, &env).Force()
		assert.Equal(t, permissionErrorOutputBinaryStream(&s), err)
		assert.False(t, ok)
	})
}

type mockTerm struct {
	mock.Mock
}

func (m *mockTerm) String() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockTerm) WriteTerm(w io.Writer, opts WriteTermOptions, env Env) error {
	args := m.Called(w, opts, env)
	return args.Error(0)
}

func (m *mockTerm) Unify(t Term, occursCheck bool, env *Env) bool {
	args := m.Called(t, occursCheck, env)
	return args.Bool(0)
}

func (m *mockTerm) Copy() Term {
	args := m.Called()
	return args.Get(0).(Term)
}

func TestCharCode(t *testing.T) {
	t.Run("ascii", func(t *testing.T) {
		env := Env{}
		ok, err := CharCode(Atom("a"), Integer(97), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("emoji", func(t *testing.T) {
		env := Env{}
		ok, err := CharCode(Atom("😀"), Integer(128512), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("query char", func(t *testing.T) {
		env := Env{}
		v := Variable("Char")

		ok, err := CharCode(v, Integer(128512), func(env Env) Promise {
			assert.Equal(t, Atom("😀"), env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("query code", func(t *testing.T) {
		env := Env{}
		v := Variable("Code")
		ok, err := CharCode(Atom("😀"), v, func(env Env) Promise {
			assert.Equal(t, Integer(128512), env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("char and code are variables", func(t *testing.T) {
		env := Env{}
		char, code := Variable("Char"), Variable("Code")

		ok, err := CharCode(char, code, Success, &env).Force()
		assert.Equal(t, instantiationError(&Compound{
			Functor: ",",
			Args:    []Term{char, code},
		}), err)
		assert.False(t, ok)
	})

	t.Run("char is neither a variable nor a one character atom", func(t *testing.T) {
		t.Run("atom", func(t *testing.T) {
			env := Env{}
			ok, err := CharCode(Atom("foo"), NewVariable(), Success, &env).Force()
			assert.Equal(t, typeErrorCharacter(Atom("foo")), err)
			assert.False(t, ok)
		})

		t.Run("non-atom", func(t *testing.T) {
			env := Env{}
			ok, err := CharCode(Integer(0), NewVariable(), Success, &env).Force()
			assert.Equal(t, typeErrorCharacter(Integer(0)), err)
			assert.False(t, ok)
		})
	})

	t.Run("code is neither a variable nor an integer", func(t *testing.T) {
		env := Env{}
		ok, err := CharCode(NewVariable(), Atom("foo"), Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Atom("foo")), err)
		assert.False(t, ok)
	})

	t.Run("code is neither a variable nor a character-code", func(t *testing.T) {
		env := Env{}
		ok, err := CharCode(NewVariable(), Integer(-1), Success, &env).Force()
		assert.Equal(t, representationError(Atom("character_code"), Atom(fmt.Sprintf("-1 is not a valid unicode code point."))), err)
		assert.False(t, ok)
	})
}

func TestVM_PutByte(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		env := Env{}

		var w mockWriter
		w.On("Write", []byte{97}).Return(1, nil).Once()
		defer w.AssertExpectations(t)

		s := Stream{sink: &w, streamType: streamTypeBinary}

		var vm VM
		ok, err := vm.PutByte(&s, Integer(97), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("ng", func(t *testing.T) {
		env := Env{}

		var w mockWriter
		w.On("Write", []byte{97}).Return(0, errors.New("")).Once()
		defer w.AssertExpectations(t)

		s := Stream{sink: &w, streamType: streamTypeBinary}

		var vm VM
		_, err := vm.PutByte(&s, Integer(97), Success, &env).Force()
		assert.Error(t, err)
	})

	t.Run("valid stream alias", func(t *testing.T) {
		env := Env{}

		var w mockWriter
		w.On("Write", []byte{97}).Return(1, nil).Once()
		defer w.AssertExpectations(t)

		s := Stream{sink: &w, streamType: streamTypeBinary}

		vm := VM{
			streams: map[Term]*Stream{
				Atom("foo"): &s,
			},
		}
		ok, err := vm.PutByte(Atom("foo"), Integer(97), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("streamOrAlias is a variable", func(t *testing.T) {
		env := Env{}
		streamOrAlias := Variable("Stream")

		var vm VM
		ok, err := vm.PutByte(streamOrAlias, Integer(97), Success, &env).Force()
		assert.Equal(t, instantiationError(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("byt is a variable", func(t *testing.T) {
		env := Env{}
		byt := Variable("Byte")

		var vm VM
		ok, err := vm.PutByte(&Stream{sink: &mockWriter{}, streamType: streamTypeBinary}, byt, Success, &env).Force()
		assert.Equal(t, instantiationError(byt), err)
		assert.False(t, ok)
	})

	t.Run("byt is neither a variable nor an byte", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.PutByte(&Stream{sink: &mockWriter{}, streamType: streamTypeBinary}, Atom("byte"), Success, &env).Force()
		assert.Equal(t, typeErrorByte(Atom("byte")), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is neither a variable nor a stream term or alias", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.PutByte(Integer(0), Integer(97), Success, &env).Force()
		assert.Equal(t, domainErrorStreamOrAlias(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is an input stream", func(t *testing.T) {
		s := Variable("Stream")
		env := Env{
			{
				Variable: s,
				Value:    &Stream{source: &mockReader{}},
			},
		}

		var vm VM
		ok, err := vm.PutByte(s, Integer(97), Success, &env).Force()
		assert.Equal(t, permissionErrorOutputStream(s), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is associated with a text stream", func(t *testing.T) {
		s := Variable("Stream")
		env := Env{
			{
				Variable: s,
				Value:    &Stream{sink: &mockWriter{}, streamType: streamTypeText},
			},
		}

		var vm VM
		ok, err := vm.PutByte(s, Integer(97), Success, &env).Force()
		assert.Equal(t, permissionErrorOutputTextStream(s), err)
		assert.False(t, ok)
	})
}

func TestVM_PutCode(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		env := Env{}

		var w mockWriter
		w.On("Write", []byte{0xf0, 0x9f, 0x98, 0x80}).Return(1, nil).Once()
		defer w.AssertExpectations(t)

		s := Stream{sink: &w}

		var vm VM
		ok, err := vm.PutCode(&s, Integer('😀'), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("ng", func(t *testing.T) {
		env := Env{}

		var w mockWriter
		w.On("Write", []byte{0xf0, 0x9f, 0x98, 0x80}).Return(0, errors.New("")).Once()
		defer w.AssertExpectations(t)

		s := Stream{sink: &w}

		var vm VM
		_, err := vm.PutCode(&s, Integer('😀'), Success, &env).Force()
		assert.Error(t, err)
	})

	t.Run("valid stream alias", func(t *testing.T) {
		env := Env{}

		var w mockWriter
		w.On("Write", []byte{0xf0, 0x9f, 0x98, 0x80}).Return(1, nil).Once()
		defer w.AssertExpectations(t)

		s := Stream{sink: &w}

		vm := VM{
			streams: map[Term]*Stream{
				Atom("foo"): &s,
			},
		}
		ok, err := vm.PutCode(Atom("foo"), Integer('😀'), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("streamOrAlias is a variable", func(t *testing.T) {
		env := Env{}
		streamOrAlias := Variable("Stream")

		var vm VM
		ok, err := vm.PutCode(streamOrAlias, Integer(97), Success, &env).Force()
		assert.Equal(t, instantiationError(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("code is a variable", func(t *testing.T) {
		env := Env{}
		code := Variable("Code")

		var vm VM
		ok, err := vm.PutCode(&Stream{sink: &mockWriter{}}, code, Success, &env).Force()
		assert.Equal(t, instantiationError(code), err)
		assert.False(t, ok)
	})

	t.Run("code is neither a variable nor an integer", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.PutCode(&Stream{sink: &mockWriter{}}, Atom("code"), Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Atom("code")), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is neither a variable nor a stream term or alias", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.PutCode(Integer(0), Integer(97), Success, &env).Force()
		assert.Equal(t, domainErrorStreamOrAlias(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is not associated with an open stream", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.PutCode(Atom("foo"), Integer(97), Success, &env).Force()
		assert.Equal(t, existenceErrorStream(Atom("foo")), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is an input stream", func(t *testing.T) {
		s := Variable("Stream")
		env := Env{
			{
				Variable: s,
				Value:    &Stream{source: &mockReader{}},
			},
		}

		var vm VM
		ok, err := vm.PutCode(s, Integer(97), Success, &env).Force()
		assert.Equal(t, permissionErrorOutputStream(s), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is associated with a binary stream", func(t *testing.T) {
		s := Variable("Stream")
		env := Env{
			{
				Variable: s,
				Value:    &Stream{sink: &mockWriter{}, streamType: streamTypeBinary},
			},
		}

		var vm VM
		ok, err := vm.PutCode(s, Integer(97), Success, &env).Force()
		assert.Equal(t, permissionErrorOutputBinaryStream(s), err)
		assert.False(t, ok)
	})

	t.Run("code is an integer but not an character code", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.PutCode(&Stream{sink: &mockWriter{}}, Integer(-1), Success, &env).Force()
		assert.Equal(t, representationError(Atom("character_code"), Atom("-1 is not a valid unicode code point.")), err)
		assert.False(t, ok)
	})

	t.Run("unknown stream alias", func(t *testing.T) {
		env := Env{}
		var vm VM
		_, err := vm.PutCode(Atom("foo"), Integer('😀'), Success, &env).Force()
		assert.Error(t, err)
	})

	t.Run("not a stream", func(t *testing.T) {
		env := Env{}
		var vm VM
		_, err := vm.PutCode(NewVariable(), Integer('😀'), Success, &env).Force()
		assert.Error(t, err)
	})

	t.Run("not a code", func(t *testing.T) {
		var w mockWriter
		defer w.AssertExpectations(t)

		s := Stream{sink: &w}

		t.Run("not an integer", func(t *testing.T) {
			env := Env{}
			var vm VM
			_, err := vm.PutCode(&s, Atom("a"), Success, &env).Force()
			assert.Error(t, err)
		})
	})
}

func TestVM_ReadTerm(t *testing.T) {
	t.Run("stream", func(t *testing.T) {
		env := Env{}
		v := Variable("Term")

		var vm VM
		ok, err := vm.ReadTerm(&Stream{source: bufio.NewReader(strings.NewReader("foo."))}, v, List(), func(env Env) Promise {
			assert.Equal(t, Atom("foo"), env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("valid stream alias", func(t *testing.T) {
		env := Env{}
		v := Variable("Term")

		s := Stream{source: bufio.NewReader(strings.NewReader("foo."))}

		vm := VM{
			streams: map[Term]*Stream{
				Atom("foo"): &s,
			},
		}
		ok, err := vm.ReadTerm(Atom("foo"), v, List(), func(env Env) Promise {
			assert.Equal(t, Atom("foo"), env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("singletons", func(t *testing.T) {
		env := Env{}
		term, singletons := Variable("Term"), Variable("Singletons")

		var vm VM
		ok, err := vm.ReadTerm(&Stream{source: bufio.NewReader(strings.NewReader("f(X, X, Y)."))}, term, List(&Compound{
			Functor: "singletons",
			Args:    []Term{singletons},
		}), func(env Env) Promise {
			assert.Equal(t, &Compound{
				Functor: "f",
				Args: []Term{
					Variable("X"),
					Variable("X"),
					Variable("Y"),
				},
			}, env.Resolve(term))

			assert.Equal(t, List(Variable("Y")), env.Resolve(singletons))

			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("variables", func(t *testing.T) {
		env := Env{}
		term, variables := Variable("Term"), Variable("Variables")

		var vm VM
		ok, err := vm.ReadTerm(&Stream{source: bufio.NewReader(strings.NewReader("f(X, X, Y)."))}, term, List(&Compound{
			Functor: "variables",
			Args:    []Term{variables},
		}), func(env Env) Promise {
			assert.Equal(t, &Compound{
				Functor: "f",
				Args: []Term{
					Variable("X"),
					Variable("X"),
					Variable("Y"),
				},
			}, env.Resolve(term))

			assert.Equal(t, List(Variable("X"), Variable("Y")), env.Resolve(variables))

			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("variable_names", func(t *testing.T) {
		env := Env{}
		term, variableNames := Variable("Term"), Variable("VariableNames")

		var vm VM
		ok, err := vm.ReadTerm(&Stream{source: bufio.NewReader(strings.NewReader("f(X, X, Y)."))}, term, List(&Compound{
			Functor: "variable_names",
			Args:    []Term{variableNames},
		}), func(env Env) Promise {
			assert.Equal(t, &Compound{
				Functor: "f",
				Args: []Term{
					Variable("X"),
					Variable("X"),
					Variable("Y"),
				},
			}, env.Resolve(term))

			assert.Equal(t, List(
				&Compound{
					Functor: "=",
					Args:    []Term{Atom("X"), Variable("X")},
				},
				&Compound{
					Functor: "=",
					Args:    []Term{Atom("Y"), Variable("Y")},
				},
			), env.Resolve(variableNames))

			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("multiple reads", func(t *testing.T) {
		s := Stream{source: bufio.NewReader(strings.NewReader(`
foo(a).
foo(b).
foo(c).
`))}

		v := Variable("Term")

		var vm VM

		env := Env{}
		ok, err := vm.ReadTerm(&s, v, List(), func(env Env) Promise {
			assert.Equal(t, &Compound{Functor: "foo", Args: []Term{Atom("a")}}, env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		env = Env{}
		ok, err = vm.ReadTerm(&s, v, List(), func(env Env) Promise {
			assert.Equal(t, &Compound{Functor: "foo", Args: []Term{Atom("b")}}, env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		env = Env{}
		ok, err = vm.ReadTerm(&s, &v, List(), func(env Env) Promise {
			assert.Equal(t, &Compound{Functor: "foo", Args: []Term{Atom("c")}}, env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("streamOrAlias is a variable", func(t *testing.T) {
		env := Env{}
		streamOrAlias := Variable("Stream")

		var vm VM
		ok, err := vm.ReadTerm(streamOrAlias, NewVariable(), List(), Success, &env).Force()
		assert.Equal(t, instantiationError(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("options is a partial list or a list with an element which is a variable", func(t *testing.T) {
		t.Run("partial list", func(t *testing.T) {
			env := Env{}

			options := ListRest(Variable("Rest"),
				&Compound{Functor: "variables", Args: []Term{Variable("VL")}},
			)

			var vm VM
			ok, err := vm.ReadTerm(&Stream{source: &mockReader{}}, NewVariable(), options, Success, &env).Force()
			assert.Equal(t, instantiationError(options), err)
			assert.False(t, ok)
		})

		t.Run("variable element", func(t *testing.T) {
			env := Env{}
			option := Variable("Option")

			var vm VM
			ok, err := vm.ReadTerm(&Stream{source: &mockReader{}}, NewVariable(), List(option, &Compound{Functor: "variables", Args: []Term{Variable("VL")}}), Success, &env).Force()
			assert.Equal(t, instantiationError(option), err)
			assert.False(t, ok)
		})
	})

	t.Run("streamOrAlias is neither a variable nor a stream term or alias", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.ReadTerm(Integer(0), NewVariable(), List(), Success, &env).Force()
		assert.Equal(t, domainErrorStreamOrAlias(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("options is neither a partial list nor a list", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.ReadTerm(&Stream{source: &mockReader{}}, NewVariable(), Atom("options"), Success, &env).Force()
		assert.Equal(t, typeErrorList(Atom("options")), err)
		assert.False(t, ok)
	})

	t.Run("an element E of the Options list is neither a variable nor a valid read-option", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.ReadTerm(&Stream{source: &mockReader{}}, NewVariable(), List(&Compound{
			Functor: "unknown",
			Args:    []Term{Atom("option")},
		}), Success, &env).Force()
		assert.Equal(t, domainErrorReadOption(&Compound{
			Functor: "unknown",
			Args:    []Term{Atom("option")},
		}), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is not associated with an open stream", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.ReadTerm(Atom("foo"), NewVariable(), List(), Success, &env).Force()
		assert.Equal(t, existenceErrorStream(Atom("foo")), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is an output stream", func(t *testing.T) {
		s := Variable("Stream")
		env := Env{
			{
				Variable: s,
				Value:    &Stream{sink: &mockWriter{}},
			},
		}

		var vm VM
		ok, err := vm.ReadTerm(s, NewVariable(), List(), Success, &env).Force()
		assert.Equal(t, permissionErrorInputStream(s), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is associated with a binary stream", func(t *testing.T) {
		s := Variable("Stream")
		env := Env{
			{
				Variable: s,
				Value:    &Stream{source: bufio.NewReader(&mockReader{}), streamType: streamTypeBinary},
			},
		}

		var vm VM
		ok, err := vm.ReadTerm(s, NewVariable(), List(), Success, &env).Force()
		assert.Equal(t, permissionErrorInputBinaryStream(s), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias has stream properties end_of_stream(past) and eof_action(error)", func(t *testing.T) {
		var r mockReader
		r.On("Read", mock.Anything).Return(0, io.EOF)

		s := Variable("Stream")
		env := Env{
			{
				Variable: s,
				Value: &Stream{
					source:    bufio.NewReader(&r),
					eofAction: eofActionError,
				},
			},
		}

		var vm VM
		ok, err := vm.ReadTerm(s, NewVariable(), List(), Success, &env).Force()
		assert.Equal(t, permissionErrorInputPastEndOfStream(s), err)
		assert.False(t, ok)
	})

	t.Run("one or more characters were input, but they cannot be parsed as a sequence of tokens", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.ReadTerm(&Stream{source: bufio.NewReader(strings.NewReader("foo bar baz."))}, NewVariable(), List(), Success, &env).Force()
		assert.Equal(t, syntaxErrorUnexpectedChar(Atom("unexpected char: b")), err)
		assert.False(t, ok)
	})

	t.Run("the sequence of tokens cannot be parsed as a term using the current set of operator definitions", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.ReadTerm(&Stream{source: bufio.NewReader(strings.NewReader("X = a."))}, NewVariable(), List(), Success, &env).Force()
		assert.Equal(t, syntaxErrorUnexpectedChar(Atom("unexpected char: =")), err)
		assert.False(t, ok)
	})
}

func TestVM_GetByte(t *testing.T) {
	t.Run("stream", func(t *testing.T) {
		s := Stream{source: strings.NewReader("a"), streamType: streamTypeBinary}

		env := Env{}
		v := Variable("Byte")

		var vm VM
		ok, err := vm.GetByte(&s, v, func(env Env) Promise {
			assert.Equal(t, Integer(97), env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("valid stream alias", func(t *testing.T) {
		s := Stream{source: strings.NewReader("a"), streamType: streamTypeBinary}

		env := Env{}
		v := Variable("Byte")

		vm := VM{
			streams: map[Term]*Stream{
				Atom("foo"): &s,
			},
		}
		ok, err := vm.GetByte(Atom("foo"), v, func(env Env) Promise {
			assert.Equal(t, Integer(97), env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("eof", func(t *testing.T) {
		s := Stream{source: strings.NewReader(""), streamType: streamTypeBinary}

		env := Env{}
		v := Variable("Byte")

		var vm VM
		ok, err := vm.GetByte(&s, v, func(env Env) Promise {
			assert.Equal(t, Integer(-1), env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("error", func(t *testing.T) {
		env := Env{}

		var m mockReader
		m.On("Read", make([]byte, 1)).Return(0, errors.New("failed")).Once()
		defer m.AssertExpectations(t)

		s := Stream{source: &m, streamType: streamTypeBinary}

		var vm VM

		v := Variable("V")
		_, err := vm.GetByte(&s, v, Success, &env).Force()
		assert.Error(t, err)
	})

	t.Run("streamOrAlias is a variable", func(t *testing.T) {
		env := Env{}
		streamOrAlias := Variable("Stream")
		var vm VM
		ok, err := vm.GetByte(streamOrAlias, Variable("InByte"), Success, &env).Force()
		assert.Equal(t, instantiationError(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("inByte is neither a variable nor an in-byte", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.GetByte(&Stream{source: &mockReader{}, streamType: streamTypeBinary}, Atom("inByte"), Success, &env).Force()
		assert.Equal(t, typeErrorInByte(Atom("inByte")), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is neither a variable nor a stream-term or alias", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.GetByte(Integer(0), Variable("InByte"), Success, &env).Force()
		assert.Equal(t, domainErrorStreamOrAlias(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is not associated with an open stream", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.GetByte(Atom("foo"), Variable("InByte"), Success, &env).Force()
		assert.Equal(t, existenceErrorStream(Atom("foo")), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is an output stream", func(t *testing.T) {
		streamOrAlias := Variable("Stream")
		env := Env{
			{
				Variable: streamOrAlias,
				Value:    &Stream{sink: &mockWriter{}},
			},
		}

		var vm VM
		ok, err := vm.GetByte(streamOrAlias, Variable("InByte"), Success, &env).Force()
		assert.Equal(t, permissionErrorInputStream(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is associated with a text stream", func(t *testing.T) {
		streamOrAlias := Variable("Stream")
		env := Env{
			{
				Variable: streamOrAlias,
				Value:    &Stream{source: &mockReader{}},
			},
		}

		var vm VM
		ok, err := vm.GetByte(streamOrAlias, Variable("InByte"), Success, &env).Force()
		assert.Equal(t, permissionErrorInputTextStream(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias has stream properties end_of_stream(past) and eof_action(error)", func(t *testing.T) {
		var r mockReader
		r.On("Read", mock.Anything).Return(0, io.EOF)

		streamOrAlias := Variable("Stream")
		env := Env{
			{
				Variable: streamOrAlias,
				Value: &Stream{
					source:     &r,
					streamType: streamTypeBinary,
					eofAction:  eofActionError,
				},
			},
		}

		var vm VM
		ok, err := vm.GetByte(streamOrAlias, Variable("InByte"), Success, &env).Force()
		assert.Equal(t, permissionErrorInputPastEndOfStream(streamOrAlias), err)
		assert.False(t, ok)
	})
}

func TestVM_GetChar(t *testing.T) {
	t.Run("stream", func(t *testing.T) {
		s := Stream{source: bufio.NewReader(strings.NewReader("😀"))}

		env := Env{}
		v := Variable("Char")

		var vm VM
		ok, err := vm.GetChar(&s, v, func(env Env) Promise {
			assert.Equal(t, Atom("😀"), env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("valid stream alias", func(t *testing.T) {
		s := Stream{source: bufio.NewReader(strings.NewReader("😀"))}

		env := Env{}
		v := Variable("Char")

		vm := VM{
			streams: map[Term]*Stream{
				Atom("foo"): &s,
			},
		}
		ok, err := vm.GetChar(Atom("foo"), v, func(env Env) Promise {
			assert.Equal(t, Atom("😀"), env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("non buffered stream", func(t *testing.T) {
		s := Variable("Stream")
		env := Env{
			{
				Variable: s,
				Value:    &Stream{source: strings.NewReader("")},
			},
		}

		var vm VM
		ok, err := vm.GetChar(s, NewVariable(), Success, &env).Force()
		assert.Equal(t, permissionErrorInputBufferedStream(s), err)
		assert.False(t, ok)
	})

	t.Run("eof", func(t *testing.T) {
		s := Stream{source: bufio.NewReader(strings.NewReader(""))}

		env := Env{}
		v := Variable("Char")

		var vm VM
		ok, err := vm.GetChar(&s, v, func(env Env) Promise {
			assert.Equal(t, Atom("end_of_file"), env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("error", func(t *testing.T) {
		env := Env{}

		var m mockReader
		m.On("Read", mock.Anything).Return(0, errors.New("failed")).Once()
		defer m.AssertExpectations(t)

		s := Stream{source: bufio.NewReader(&m)}

		v := Variable("V")

		var vm VM
		ok, err := vm.GetChar(&s, v, Success, &env).Force()
		assert.Equal(t, systemError(errors.New("failed")), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is a variable", func(t *testing.T) {
		env := Env{}
		streamOrAlias := Variable("Stream")

		var vm VM
		ok, err := vm.GetChar(streamOrAlias, Variable("Char"), Success, &env).Force()
		assert.Equal(t, instantiationError(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("char is neither a variable nor an in-character", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.GetChar(&Stream{source: bufio.NewReader(&mockReader{})}, Integer(0), Success, &env).Force()
		assert.Equal(t, typeErrorInCharacter(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is neither a variable nor a stream term or alias", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.GetChar(Integer(0), Variable("Char"), Success, &env).Force()
		assert.Equal(t, domainErrorStreamOrAlias(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is an output stream", func(t *testing.T) {
		streamOrAlias := Variable("Stream")
		env := Env{
			{
				Variable: streamOrAlias,
				Value:    &Stream{sink: &mockWriter{}},
			},
		}

		var vm VM
		ok, err := vm.GetChar(streamOrAlias, Variable("Char"), Success, &env).Force()
		assert.Equal(t, permissionErrorInputStream(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is associated with a binary stream", func(t *testing.T) {
		streamOrAlias := Variable("Stream")
		env := Env{
			{
				Variable: streamOrAlias,
				Value:    &Stream{source: bufio.NewReader(&mockReader{}), streamType: streamTypeBinary},
			},
		}

		var vm VM
		ok, err := vm.GetChar(streamOrAlias, Variable("Char"), Success, &env).Force()
		assert.Equal(t, permissionErrorInputBinaryStream(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias has stream properties end_of_stream(past) and eof_action(error)", func(t *testing.T) {
		var r mockReader
		r.On("Read", mock.Anything).Return(0, io.EOF)

		streamOrAlias := Variable("Stream")
		env := Env{
			{
				Variable: streamOrAlias,
				Value: &Stream{
					source:    bufio.NewReader(&r),
					eofAction: eofActionError,
				},
			},
		}

		var vm VM
		ok, err := vm.GetChar(streamOrAlias, Variable("Char"), Success, &env).Force()
		assert.Equal(t, permissionErrorInputPastEndOfStream(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("the entity input from the stream is not a character", func(t *testing.T) {
		streamOrAlias := Variable("Stream")
		env := Env{
			{
				Variable: streamOrAlias,
				Value:    &Stream{source: bufio.NewReader(bytes.NewBufferString(string(unicode.ReplacementChar)))},
			},
		}

		var vm VM
		ok, err := vm.GetChar(streamOrAlias, Variable("Char"), Success, &env).Force()
		assert.Equal(t, representationError(Atom("character"), Atom("invalid character.")), err)
		assert.False(t, ok)
	})
}

func TestVM_PeekByte(t *testing.T) {
	t.Run("stream", func(t *testing.T) {
		s := Stream{source: bufio.NewReader(strings.NewReader("abc")), streamType: streamTypeBinary}

		env := Env{}
		v := Variable("Byte")

		var vm VM
		ok, err := vm.PeekByte(&s, v, func(env Env) Promise {
			assert.Equal(t, Integer(97), env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = vm.PeekByte(&s, v, Success, &env).Force() // 'a' again
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("valid stream alias", func(t *testing.T) {
		s := Stream{source: bufio.NewReader(strings.NewReader("abc")), streamType: streamTypeBinary}

		env := Env{}
		v := Variable("Byte")

		vm := VM{
			streams: map[Term]*Stream{
				Atom("foo"): &s,
			},
		}
		ok, err := vm.PeekByte(Atom("foo"), v, func(env Env) Promise {
			assert.Equal(t, Integer(97), env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("non buffered stream", func(t *testing.T) {
		s := Variable("Stream")
		env := Env{
			{
				Variable: s,
				Value:    &Stream{source: strings.NewReader(""), streamType: streamTypeBinary},
			},
		}

		var vm VM
		ok, err := vm.PeekByte(s, NewVariable(), Success, &env).Force()
		assert.Equal(t, permissionErrorInputBufferedStream(s), err)
		assert.False(t, ok)
	})

	t.Run("eof", func(t *testing.T) {
		s := Stream{source: bufio.NewReader(strings.NewReader("")), streamType: streamTypeBinary}

		env := Env{}
		v := Variable("Byte")

		var vm VM
		ok, err := vm.PeekByte(&s, v, func(env Env) Promise {
			assert.Equal(t, Integer(-1), env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("error", func(t *testing.T) {
		env := Env{}

		var m mockReader
		m.On("Read", mock.Anything).Return(0, errors.New("failed")).Once()
		defer m.AssertExpectations(t)

		s := Stream{source: bufio.NewReader(&m), streamType: streamTypeBinary}

		v := Variable("V")

		var vm VM
		ok, err := vm.PeekByte(&s, v, Success, &env).Force()
		assert.Equal(t, systemError(errors.New("failed")), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is a variable", func(t *testing.T) {
		env := Env{}
		streamOrAlias := Variable("Stream")

		var vm VM
		ok, err := vm.PeekByte(streamOrAlias, Variable("Byte"), Success, &env).Force()
		assert.Equal(t, instantiationError(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("inByte is neither a variable nor an in-byte", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.PeekByte(&Stream{source: bufio.NewReader(&mockReader{}), streamType: streamTypeBinary}, Atom("byte"), Success, &env).Force()
		assert.Equal(t, typeErrorInByte(Atom("byte")), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is neither a variable nor a stream term or alias", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.PeekByte(Integer(0), Variable("Byte"), Success, &env).Force()
		assert.Equal(t, domainErrorStreamOrAlias(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is an output stream", func(t *testing.T) {
		streamOrAlias := Variable("Stream")
		env := Env{
			{
				Variable: streamOrAlias,
				Value:    &Stream{sink: &mockWriter{}},
			},
		}

		var vm VM
		ok, err := vm.PeekByte(streamOrAlias, Variable("Byte"), Success, &env).Force()
		assert.Equal(t, permissionErrorInputStream(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is associated with a text stream", func(t *testing.T) {
		streamOrAlias := Variable("Stream")
		env := Env{
			{
				Variable: streamOrAlias,
				Value:    &Stream{source: bufio.NewReader(&mockReader{}), streamType: streamTypeText},
			},
		}

		var vm VM
		ok, err := vm.PeekByte(streamOrAlias, Variable("Byte"), Success, &env).Force()
		assert.Equal(t, permissionErrorInputTextStream(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias has stream properties end_of_stream(past) and eof_action(error)", func(t *testing.T) {
		var r mockReader
		r.On("Read", mock.Anything).Return(0, io.EOF)

		streamOrAlias := Variable("Stream")
		env := Env{
			{
				Variable: streamOrAlias,
				Value: &Stream{
					source:     bufio.NewReader(&r),
					streamType: streamTypeBinary,
					eofAction:  eofActionError,
				},
			},
		}

		var vm VM
		ok, err := vm.PeekByte(streamOrAlias, Variable("Byte"), Success, &env).Force()
		assert.Equal(t, permissionErrorInputPastEndOfStream(streamOrAlias), err)
		assert.False(t, ok)
	})
}

func TestVM_PeekChar(t *testing.T) {
	t.Run("stream", func(t *testing.T) {
		s := Stream{source: bufio.NewReader(strings.NewReader("😀❗"))}

		env := Env{}
		v := Variable("Char")

		var vm VM
		ok, err := vm.PeekChar(&s, v, func(env Env) Promise {
			assert.Equal(t, Atom("😀"), env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = vm.PeekChar(&s, v, Success, &env).Force() // '😀' again
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("valid stream alias", func(t *testing.T) {
		s := Stream{source: bufio.NewReader(strings.NewReader("😀❗"))}

		env := Env{}
		v := Variable("Char")

		vm := VM{
			streams: map[Term]*Stream{
				Atom("foo"): &s,
			},
		}
		ok, err := vm.PeekChar(Atom("foo"), v, func(env Env) Promise {
			assert.Equal(t, Atom("😀"), env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("non buffered stream", func(t *testing.T) {
		s := Variable("Stream")
		env := Env{
			{
				Variable: s,
				Value:    &Stream{source: strings.NewReader("")},
			},
		}

		var vm VM
		ok, err := vm.PeekChar(s, NewVariable(), Success, &env).Force()
		assert.Equal(t, permissionErrorInputBufferedStream(s), err)
		assert.False(t, ok)
	})

	t.Run("eof", func(t *testing.T) {
		s := Stream{source: bufio.NewReader(strings.NewReader(""))}

		env := Env{}
		v := Variable("Char")

		var vm VM
		ok, err := vm.PeekChar(&s, v, func(env Env) Promise {
			assert.Equal(t, Atom("end_of_file"), env.Resolve(v))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("error", func(t *testing.T) {
		env := Env{}

		var m mockReader
		m.On("Read", mock.Anything).Return(0, errors.New("failed")).Once()
		defer m.AssertExpectations(t)

		s := Stream{source: bufio.NewReader(&m)}

		v := Variable("V")

		var vm VM
		ok, err := vm.PeekChar(&s, v, Success, &env).Force()
		assert.Equal(t, systemError(errors.New("failed")), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is a variable", func(t *testing.T) {
		env := Env{}
		streamOrAlias := Variable("Stream")

		var vm VM
		ok, err := vm.PeekChar(streamOrAlias, Variable("Char"), Success, &env).Force()
		assert.Equal(t, instantiationError(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("char is neither a variable nor an in-character", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.PeekChar(&Stream{source: bufio.NewReader(&mockReader{})}, Integer(0), Success, &env).Force()
		assert.Equal(t, typeErrorInCharacter(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is neither a variable nor a stream term or alias", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.PeekChar(Integer(0), Variable("Char"), Success, &env).Force()
		assert.Equal(t, domainErrorStreamOrAlias(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is an output stream", func(t *testing.T) {
		streamOrAlias := Variable("Stream")
		env := Env{
			{
				Variable: streamOrAlias,
				Value:    &Stream{sink: &mockWriter{}},
			},
		}

		var vm VM
		ok, err := vm.PeekChar(streamOrAlias, Variable("Char"), Success, &env).Force()
		assert.Equal(t, permissionErrorInputStream(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is associated with a binary stream", func(t *testing.T) {
		streamOrAlias := Variable("Stream")
		env := Env{
			{
				Variable: streamOrAlias,
				Value: &Stream{
					source:     bufio.NewReader(&mockReader{}),
					streamType: streamTypeBinary,
				},
			},
		}

		var vm VM
		ok, err := vm.PeekChar(streamOrAlias, Variable("Char"), Success, &env).Force()
		assert.Equal(t, permissionErrorInputBinaryStream(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias has stream properties end_of_stream(past) and eof_action(error)", func(t *testing.T) {
		var r mockReader
		r.On("Read", mock.Anything).Return(0, io.EOF)

		streamOrAlias := Variable("Stream")
		env := Env{
			{
				Variable: streamOrAlias,
				Value: &Stream{
					source:    bufio.NewReader(&r),
					eofAction: eofActionError,
				},
			},
		}

		var vm VM
		ok, err := vm.PeekChar(streamOrAlias, Variable("Char"), Success, &env).Force()
		assert.Equal(t, permissionErrorInputPastEndOfStream(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("the entity input from the stream is not a character", func(t *testing.T) {
		streamOrAlias := Variable("Stream")
		env := Env{
			{
				Variable: streamOrAlias,
				Value:    &Stream{source: bufio.NewReader(bytes.NewBufferString(string(unicode.ReplacementChar)))},
			},
		}

		var vm VM
		ok, err := vm.PeekChar(streamOrAlias, Variable("Char"), Success, &env).Force()
		assert.Equal(t, representationError(Atom("character"), Atom("invalid character.")), err)
		assert.False(t, ok)
	})
}

func TestVM_Halt(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		env := Env{}
		var exitCalled bool
		osExit = func(code int) {
			assert.Equal(t, 2, code)
			exitCalled = true
		}
		defer func() {
			osExit = os.Exit
		}()

		var callbackCalled bool
		vm := VM{
			OnHalt: func() {
				callbackCalled = true
			},
		}
		ok, err := vm.Halt(Integer(2), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		assert.True(t, exitCalled)
		assert.True(t, callbackCalled)
	})

	t.Run("n is a variable", func(t *testing.T) {
		env := Env{}
		n := Variable("N")

		var vm VM
		ok, err := vm.Halt(n, Success, &env).Force()
		assert.Equal(t, instantiationError(n), err)
		assert.False(t, ok)
	})

	t.Run("n is neither a variable nor an integer", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Halt(Atom("foo"), Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Atom("foo")), err)
		assert.False(t, ok)
	})
}

func TestVM_Clause(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		env := Env{}
		x := Variable("X")
		what, body := Variable("What"), Variable("Body")

		var c int

		vm := VM{
			procedures: map[procedureIndicator]procedure{
				{name: "green", arity: 1}: clauses{
					{raw: &Compound{
						Functor: ":-", Args: []Term{
							&Compound{Functor: "green", Args: []Term{x}},
							&Compound{Functor: "moldy", Args: []Term{x}},
						},
					}},
					{raw: &Compound{Functor: "green", Args: []Term{Atom("kermit")}}},
				},
			},
		}
		ok, err := vm.Clause(&Compound{
			Functor: "green",
			Args:    []Term{what},
		}, body, func(env Env) Promise {
			switch c {
			case 0:
				assert.True(t, env.Resolve(what).(Variable).Anonymous())
				b, ok := env.Resolve(body).(*Compound)
				assert.True(t, ok)
				assert.Equal(t, Atom("moldy"), b.Functor)
				assert.Len(t, b.Args, 1)
				assert.True(t, b.Args[0].(Variable).Anonymous())
			case 1:
				assert.Equal(t, Atom("kermit"), env.Resolve(what))
				assert.Equal(t, Atom("true"), env.Resolve(body))
			default:
				assert.Fail(t, "unreachable")
			}
			c++
			return Bool(false)
		}, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("head is a variable", func(t *testing.T) {
		env := Env{}
		head := Variable("Head")

		var vm VM
		ok, err := vm.Clause(head, Atom("true"), Success, &env).Force()
		assert.Equal(t, instantiationError(head), err)
		assert.False(t, ok)
	})

	t.Run("head is neither a variable nor a predication", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Clause(Integer(0), Atom("true"), Success, &env).Force()
		assert.Equal(t, typeErrorCallable(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("the predicate indicator Pred of Head is that of a private (ie. Not public) procedure", func(t *testing.T) {
		// TODO: we haven't introduced the concept of private procedure yet.
	})

	t.Run("body is neither a variable nor a callable term", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.Clause(Atom("foo"), Integer(0), Success, &env).Force()
		assert.Equal(t, typeErrorCallable(Integer(0)), err)
		assert.False(t, ok)
	})
}

func TestAtomLength(t *testing.T) {
	t.Run("ascii", func(t *testing.T) {
		env := Env{}
		ok, err := AtomLength(Atom("abc"), Integer(3), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("emoji", func(t *testing.T) {
		env := Env{}
		ok, err := AtomLength(Atom("😀"), Integer(1), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("atom is a variable", func(t *testing.T) {
		env := Env{}
		atom := Variable("Atom")
		ok, err := AtomLength(atom, Integer(0), Success, &env).Force()
		assert.Equal(t, instantiationError(atom), err)
		assert.False(t, ok)
	})

	t.Run("atom is neither a variable nor an atom", func(t *testing.T) {
		env := Env{}
		ok, err := AtomLength(Integer(2), Integer(0), Success, &env).Force()
		assert.Equal(t, typeErrorAtom(Integer(2)), err)
		assert.False(t, ok)
	})

	t.Run("length is neither a variable nor an integer", func(t *testing.T) {
		env := Env{}
		ok, err := AtomLength(Atom("😀"), Atom("1"), Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Atom("1")), err)
		assert.False(t, ok)
	})

	t.Run("length is an integer less than zero", func(t *testing.T) {
		env := Env{}
		ok, err := AtomLength(Atom("😀"), Integer(-1), Success, &env).Force()
		assert.Equal(t, domainErrorNotLessThanZero(Integer(-1)), err)
		assert.False(t, ok)
	})
}

func TestAtomConcat(t *testing.T) {
	t.Run("atom3 is a variable", func(t *testing.T) {
		env := Env{}
		atom3 := Variable("Atom3")

		ok, err := AtomConcat(Atom("foo"), Atom("bar"), atom3, func(env Env) Promise {
			assert.Equal(t, Atom("foobar"), env.Resolve(atom3))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("atom3 is an atom", func(t *testing.T) {
		env := Env{}
		var c int
		v1, v2 := Variable("V1"), Variable("V2")
		ok, err := AtomConcat(v1, v2, Atom("foo"), func(env Env) Promise {
			switch c {
			case 0:
				assert.Equal(t, Atom(""), env.Resolve(v1))
				assert.Equal(t, Atom("foo"), env.Resolve(v2))
			case 1:
				assert.Equal(t, Atom("f"), env.Resolve(v1))
				assert.Equal(t, Atom("oo"), env.Resolve(v2))
			case 2:
				assert.Equal(t, Atom("fo"), env.Resolve(v1))
				assert.Equal(t, Atom("o"), env.Resolve(v2))
			case 3:
				assert.Equal(t, Atom("foo"), env.Resolve(v1))
				assert.Equal(t, Atom(""), env.Resolve(v2))
			default:
				assert.Fail(t, "unreachable")
			}
			c++
			return Bool(false)
		}, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("atom1 and atom3 are variables", func(t *testing.T) {
		env := Env{}
		atom1, atom3 := Variable("Atom1"), Variable("Atom3")

		ok, err := AtomConcat(atom1, Atom("bar"), atom3, Success, &env).Force()
		assert.Equal(t, instantiationError(&Compound{
			Functor: ",",
			Args:    []Term{atom1, atom3},
		}), err)
		assert.False(t, ok)
	})

	t.Run("atom2 and atom3 are variables", func(t *testing.T) {
		env := Env{}
		atom2, atom3 := Variable("Atom2"), Variable("Atom3")

		ok, err := AtomConcat(Atom("foo"), atom2, atom3, Success, &env).Force()
		assert.Equal(t, instantiationError(&Compound{
			Functor: ",",
			Args:    []Term{atom2, atom3},
		}), err)
		assert.False(t, ok)
	})

	t.Run("atom1 is neither a variable nor an atom", func(t *testing.T) {
		t.Run("atom3 is a variable", func(t *testing.T) {
			env := Env{}
			ok, err := AtomConcat(Integer(1), Atom("bar"), Variable("Atom3"), Success, &env).Force()
			assert.Equal(t, typeErrorAtom(Integer(1)), err)
			assert.False(t, ok)
		})

		t.Run("atom3 is an atom", func(t *testing.T) {
			env := Env{}
			ok, err := AtomConcat(Integer(1), Atom("bar"), Atom("foobar"), Success, &env).Force()
			assert.Equal(t, typeErrorAtom(Integer(1)), err)
			assert.False(t, ok)
		})
	})

	t.Run("atom2 is neither a variable nor an atom", func(t *testing.T) {
		t.Run("atom3 is a variable", func(t *testing.T) {
			env := Env{}
			ok, err := AtomConcat(Atom("foo"), Integer(2), Variable("Atom3"), Success, &env).Force()
			assert.Equal(t, typeErrorAtom(Integer(2)), err)
			assert.False(t, ok)
		})

		t.Run("atom3 is an atom", func(t *testing.T) {
			env := Env{}
			ok, err := AtomConcat(Atom("foo"), Integer(2), Atom("foobar"), Success, &env).Force()
			assert.Equal(t, typeErrorAtom(Integer(2)), err)
			assert.False(t, ok)
		})
	})

	t.Run("atom3 is neither a variable nor an atom", func(t *testing.T) {
		env := Env{}
		ok, err := AtomConcat(Atom("foo"), Atom("bar"), Integer(3), Success, &env).Force()
		assert.Equal(t, typeErrorAtom(Integer(3)), err)
		assert.False(t, ok)
	})
}

func TestSubAtom(t *testing.T) {
	t.Run("multiple solutions", func(t *testing.T) {
		env := Env{}
		before, length, after := Variable("Before"), Variable("Length"), Variable("After")
		var c int
		ok, err := SubAtom(Atom("xATGATGAxATGAxATGAx"), before, length, after, Atom("ATGA"), func(env Env) Promise {
			switch c {
			case 0:
				assert.Equal(t, Integer(1), env.Resolve(before))
				assert.Equal(t, Integer(4), env.Resolve(length))
				assert.Equal(t, Integer(14), env.Resolve(after))
			case 1:
				assert.Equal(t, Integer(4), env.Resolve(before))
				assert.Equal(t, Integer(4), env.Resolve(length))
				assert.Equal(t, Integer(11), env.Resolve(after))
			case 2:
				assert.Equal(t, Integer(9), env.Resolve(before))
				assert.Equal(t, Integer(4), env.Resolve(length))
				assert.Equal(t, Integer(6), env.Resolve(after))
			case 3:
				assert.Equal(t, Integer(14), env.Resolve(before))
				assert.Equal(t, Integer(4), env.Resolve(length))
				assert.Equal(t, Integer(1), env.Resolve(after))
			default:
				assert.Fail(t, "unreachable")
			}
			c++
			return Bool(false)
		}, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("get the first char", func(t *testing.T) {
		env := Env{}
		char := Variable("Char")
		ok, err := SubAtom(Atom("a"), Integer(0), Integer(1), Integer(0), char, func(env Env) Promise {
			assert.Equal(t, Atom("a"), env.Resolve(char))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("atom is a variable", func(t *testing.T) {
		env := Env{}
		atom := Variable("Atom")
		ok, err := SubAtom(atom, Variable("Before"), Variable("Length"), Variable("After"), Variable("SubAtom"), Success, &env).Force()
		assert.Equal(t, instantiationError(atom), err)
		assert.False(t, ok)
	})

	t.Run("atom is neither a variable nor an atom", func(t *testing.T) {
		env := Env{}
		ok, err := SubAtom(Integer(0), Variable("Before"), Variable("Length"), Variable("After"), Variable("SubAtom"), Success, &env).Force()
		assert.Equal(t, typeErrorAtom(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("subAtom is neither a variable nor an atom", func(t *testing.T) {
		env := Env{}
		ok, err := SubAtom(Atom("foo"), Variable("Before"), Variable("Length"), Variable("After"), Integer(0), Success, &env).Force()
		assert.Equal(t, typeErrorAtom(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("before is neither a variable nor an integer", func(t *testing.T) {
		env := Env{}
		ok, err := SubAtom(Atom("foo"), Atom("before"), Variable("Length"), Variable("After"), Variable("SubAtom"), Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Atom("before")), err)
		assert.False(t, ok)
	})

	t.Run("length is neither a variable nor an integer", func(t *testing.T) {
		env := Env{}
		ok, err := SubAtom(Atom("foo"), Variable("Before"), Atom("length"), Variable("After"), Variable("SubAtom"), Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Atom("length")), err)
		assert.False(t, ok)
	})

	t.Run("after is neither a variable nor an integer", func(t *testing.T) {
		env := Env{}
		ok, err := SubAtom(Atom("foo"), Variable("Before"), Variable("Length"), Atom("after"), Variable("SubAtom"), Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Atom("after")), err)
		assert.False(t, ok)
	})

	t.Run("before is an integer less than zero", func(t *testing.T) {
		env := Env{}
		ok, err := SubAtom(Atom("foo"), Integer(-1), Variable("Length"), Variable("After"), Variable("SubAtom"), Success, &env).Force()
		assert.Equal(t, domainErrorNotLessThanZero(Integer(-1)), err)
		assert.False(t, ok)
	})

	t.Run("length is an integer less than zero", func(t *testing.T) {
		env := Env{}
		ok, err := SubAtom(Atom("foo"), Variable("Before"), Integer(-1), Variable("After"), Variable("SubAtom"), Success, &env).Force()
		assert.Equal(t, domainErrorNotLessThanZero(Integer(-1)), err)
		assert.False(t, ok)
	})

	t.Run("after is an integer less than zero", func(t *testing.T) {
		env := Env{}
		ok, err := SubAtom(Atom("foo"), Variable("Before"), Variable("Length"), Integer(-1), Variable("SubAtom"), Success, &env).Force()
		assert.Equal(t, domainErrorNotLessThanZero(Integer(-1)), err)
		assert.False(t, ok)
	})
}

func TestAtomChars(t *testing.T) {
	t.Run("break down", func(t *testing.T) {
		env := Env{}
		chars := Variable("Char")

		ok, err := AtomChars(Atom("foo"), chars, func(env Env) Promise {
			assert.Equal(t, List(Atom("f"), Atom("o"), Atom("o")), env.Resolve(chars))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("construct", func(t *testing.T) {
		env := Env{}
		atom := Variable("Atom")

		ok, err := AtomChars(atom, List(Atom("f"), Atom("o"), Atom("o")), func(env Env) Promise {
			assert.Equal(t, Atom("foo"), env.Resolve(atom))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		_, err = AtomChars(NewVariable(), List(Integer(102), Integer(111), Integer(111)), Success, &env).Force()
		assert.Error(t, err)
	})

	t.Run("atom is a variable and List is a partial list or list with an element which is a variable", func(t *testing.T) {
		t.Run("partial list", func(t *testing.T) {
			env := Env{}
			chars := ListRest(Variable("Rest"),
				Atom("0"),
				Atom("0"),
			)

			ok, err := AtomChars(NewVariable(), chars, Success, &env).Force()
			assert.Equal(t, instantiationError(chars), err)
			assert.False(t, ok)
		})

		t.Run("variable element", func(t *testing.T) {
			env := Env{}
			char := Variable("Char")
			ok, err := AtomChars(NewVariable(), List(char, Atom("o"), Atom("o")), Success, &env).Force()
			assert.Equal(t, instantiationError(char), err)
			assert.False(t, ok)
		})
	})

	t.Run("atom is neither a variable nor an atom", func(t *testing.T) {
		env := Env{}
		ok, err := AtomChars(Integer(0), NewVariable(), Success, &env).Force()
		assert.Equal(t, typeErrorAtom(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("atom is a variable and List is neither a list nor a partial list", func(t *testing.T) {
		env := Env{}
		ok, err := AtomChars(NewVariable(), Atom("chars"), Success, &env).Force()
		assert.Equal(t, typeErrorList(Atom("chars")), err)
		assert.False(t, ok)
	})

	t.Run("atom is a variable and an element E of the list List is neither a variable nor a one-character atom", func(t *testing.T) {
		t.Run("not a one-character atom", func(t *testing.T) {
			env := Env{}
			ok, err := AtomChars(NewVariable(), List(Atom("chars")), Success, &env).Force()
			assert.Equal(t, typeErrorCharacter(Atom("chars")), err)
			assert.False(t, ok)
		})

		t.Run("not an atom", func(t *testing.T) {
			env := Env{}
			ok, err := AtomChars(NewVariable(), List(Integer(0)), Success, &env).Force()
			assert.Equal(t, typeErrorCharacter(Integer(0)), err)
			assert.False(t, ok)
		})
	})
}

func TestAtomCodes(t *testing.T) {
	t.Run("break up", func(t *testing.T) {
		env := Env{}
		codes := Variable("Codes")

		ok, err := AtomCodes(Atom("foo"), codes, func(env Env) Promise {
			assert.Equal(t, List(Integer(102), Integer(111), Integer(111)), env.Resolve(codes))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("construct", func(t *testing.T) {
		env := Env{}
		atom := Variable("Atom")

		ok, err := AtomCodes(atom, List(Integer(102), Integer(111), Integer(111)), func(env Env) Promise {
			assert.Equal(t, Atom("foo"), env.Resolve(atom))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("atom is a variable and List is a partial list or list with an element which is a variable", func(t *testing.T) {
		t.Run("partial list", func(t *testing.T) {
			env := Env{}
			codes := ListRest(Variable("Rest"),
				Integer(111),
				Integer(111),
			)
			ok, err := AtomCodes(NewVariable(), codes, Success, &env).Force()
			assert.Equal(t, instantiationError(codes), err)
			assert.False(t, ok)
		})

		t.Run("variable element", func(t *testing.T) {
			env := Env{}
			code := Variable("Code")

			ok, err := AtomCodes(NewVariable(), List(code, Integer(111), Integer(111)), Success, &env).Force()
			assert.Equal(t, instantiationError(code), err)
			assert.False(t, ok)
		})
	})

	t.Run("atom is neither a variable nor an atom", func(t *testing.T) {
		env := Env{}
		ok, err := AtomCodes(Integer(0), List(Integer(102), Integer(111), Integer(111)), Success, &env).Force()
		assert.Equal(t, typeErrorAtom(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("atom is a variable and List is neither a list nor a partial list", func(t *testing.T) {
		env := Env{}
		ok, err := AtomCodes(NewVariable(), Atom("codes"), Success, &env).Force()
		assert.Equal(t, typeErrorList(Atom("codes")), err)
		assert.False(t, ok)
	})

	t.Run("atom is a variable and an element E of the list List is neither a variable nor a character-code", func(t *testing.T) {
		env := Env{}
		ok, err := AtomCodes(NewVariable(), List(Atom("f"), Integer(111), Integer(111)), Success, &env).Force()
		assert.Equal(t, representationError(Atom("character_code"), Atom("invalid character code.")), err)
		assert.False(t, ok)
	})
}

func TestNumberChars(t *testing.T) {
	t.Run("number to chars", func(t *testing.T) {
		env := Env{}
		chars := Variable("Chars")

		ok, err := NumberChars(Float(23.4), chars, func(env Env) Promise {
			assert.Equal(t, List(Atom("2"), Atom("3"), Atom("."), Atom("4")), env.Resolve(chars))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("chars to number", func(t *testing.T) {
		env := Env{}
		num := Variable("Num")

		ok, err := NumberChars(num, List(Atom("2"), Atom("3"), Atom("."), Atom("4")), func(env Env) Promise {
			assert.Equal(t, Float(23.4), env.Resolve(num))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("num is a variable and chars is a partial list or list with an element which is a variable", func(t *testing.T) {
		t.Run("partial list", func(t *testing.T) {
			env := Env{}
			codes := ListRest(Variable("Rest"),
				Atom("2"), Atom("3"), Atom("."), Atom("4"),
			)

			ok, err := NumberChars(NewVariable(), codes, Success, &env).Force()
			assert.Equal(t, instantiationError(codes), err)
			assert.False(t, ok)
		})

		t.Run("variable element", func(t *testing.T) {
			env := Env{}
			code := Variable("Code")

			ok, err := NumberChars(NewVariable(), List(code, Atom("3"), Atom("."), Atom("4")), Success, &env).Force()
			assert.Equal(t, instantiationError(code), err)
			assert.False(t, ok)
		})
	})

	t.Run("num is neither a variable nor a number", func(t *testing.T) {
		env := Env{}
		ok, err := NumberChars(Atom("23.4"), List(Atom("2"), Atom("3"), Atom("."), Atom("4")), Success, &env).Force()
		assert.Equal(t, typeErrorNumber(Atom("23.4")), err)
		assert.False(t, ok)
	})

	t.Run("num is a variable and chars is neither a list nor partial list", func(t *testing.T) {
		env := Env{}
		ok, err := NumberChars(NewVariable(), Atom("23.4"), Success, &env).Force()
		assert.Equal(t, typeErrorList(Atom("23.4")), err)
		assert.False(t, ok)
	})

	t.Run("an element E of the list chars is neither a variable nor a one-character atom", func(t *testing.T) {
		env := Env{}
		ok, err := NumberChars(NewVariable(), List(Integer(2), Atom("3"), Atom("."), Atom("4")), Success, &env).Force()
		assert.Equal(t, typeErrorCharacter(Integer(2)), err)
		assert.False(t, ok)
	})

	t.Run("chars is a list of one-char atoms but is not parsable as a number", func(t *testing.T) {
		env := Env{}
		ok, err := NumberChars(NewVariable(), List(Atom("f"), Atom("o"), Atom("o")), Success, &env).Force()
		assert.Equal(t, syntaxErrorNotANumber(), err)
		assert.False(t, ok)
	})
}

func TestNumberCodes(t *testing.T) {
	t.Run("number to codes", func(t *testing.T) {
		env := Env{}
		codes := Variable("Codes")

		ok, err := NumberCodes(Float(23.4), codes, func(env Env) Promise {
			assert.Equal(t, List(Integer(50), Integer(51), Integer(46), Integer(52)), env.Resolve(codes))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("codes to number", func(t *testing.T) {
		env := Env{}
		num := Variable("Num")

		ok, err := NumberCodes(num, List(Integer(50), Integer(51), Integer(46), Integer(52)), func(env Env) Promise {
			assert.Equal(t, Float(23.4), env.Resolve(num))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("num is a variable and codes is a partial list or list with an element which is a variable", func(t *testing.T) {
		t.Run("partial list", func(t *testing.T) {
			env := Env{}
			codes := ListRest(Variable("Rest"),
				Integer(50), Integer(51), Integer(46), Integer(52),
			)

			ok, err := NumberCodes(NewVariable(), codes, Success, &env).Force()
			assert.Equal(t, instantiationError(codes), err)
			assert.False(t, ok)
		})

		t.Run("variable element", func(t *testing.T) {
			env := Env{}
			code := Variable("Code")

			ok, err := NumberCodes(NewVariable(), List(code, Integer(50), Integer(51), Integer(46), Integer(52)), Success, &env).Force()
			assert.Equal(t, instantiationError(code), err)
			assert.False(t, ok)
		})
	})

	t.Run("num is neither a variable nor a number", func(t *testing.T) {
		env := Env{}
		ok, err := NumberCodes(Atom("23.4"), List(Integer(50), Integer(51), Integer(46), Integer(52)), Success, &env).Force()
		assert.Equal(t, typeErrorNumber(Atom("23.4")), err)
		assert.False(t, ok)
	})

	t.Run("num is a variable and codes is neither a list nor partial list", func(t *testing.T) {
		env := Env{}
		ok, err := NumberCodes(NewVariable(), Atom("23.4"), Success, &env).Force()
		assert.Equal(t, typeErrorList(Atom("23.4")), err)
		assert.False(t, ok)
	})

	t.Run("an element E of the list codes is neither a variable nor a one-character atom", func(t *testing.T) {
		env := Env{}
		ok, err := NumberCodes(NewVariable(), List(Atom("2"), Integer(51), Integer(46), Integer(52)), Success, &env).Force()
		assert.Equal(t, representationError(Atom("character_code"), Atom("'2' is not a valid character code.")), err)
		assert.False(t, ok)
	})

	t.Run("codes is a list of one-char atoms but is not parsable as a number", func(t *testing.T) {
		env := Env{}
		ok, err := NumberCodes(NewVariable(), List(Integer(102), Integer(111), Integer(111)), Success, &env).Force()
		assert.Equal(t, syntaxErrorNotANumber(), err)
		assert.False(t, ok)
	})
}

func TestFunctionSet_Is(t *testing.T) {
	t.Run("addition", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Integer(3), &Compound{Functor: "+", Args: []Term{Integer(1), Integer(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(3), &Compound{Functor: "+", Args: []Term{Integer(1), Float(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(3), &Compound{Functor: "+", Args: []Term{Float(1), Integer(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(3), &Compound{Functor: "+", Args: []Term{Float(1), Float(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("subtraction", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Integer(1), &Compound{Functor: "-", Args: []Term{Integer(3), Integer(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(1), &Compound{Functor: "-", Args: []Term{Integer(3), Float(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(1), &Compound{Functor: "-", Args: []Term{Float(3), Integer(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(1), &Compound{Functor: "-", Args: []Term{Float(3), Float(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("multiplication", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Integer(6), &Compound{Functor: "*", Args: []Term{Integer(3), Integer(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(6), &Compound{Functor: "*", Args: []Term{Integer(3), Float(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(6), &Compound{Functor: "*", Args: []Term{Float(3), Integer(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(6), &Compound{Functor: "*", Args: []Term{Float(3), Float(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("floating-point division", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Float(2), &Compound{Functor: "/", Args: []Term{Integer(4), Integer(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(2), &Compound{Functor: "/", Args: []Term{Integer(4), Float(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(2), &Compound{Functor: "/", Args: []Term{Float(4), Integer(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(2), &Compound{Functor: "/", Args: []Term{Float(4), Float(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("integer division", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Integer(2), &Compound{Functor: "//", Args: []Term{Integer(4), Integer(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: "//", Args: []Term{Integer(4), Float(2)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(2)), err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: "//", Args: []Term{Float(4), Integer(2)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(4)), err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: "//", Args: []Term{Integer(4), Integer(0)}}, Success, &env).Force()
		assert.Equal(t, evaluationErrorZeroDivisor(), err)
		assert.False(t, ok)
	})

	t.Run("remainder", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Integer(-1), &Compound{Functor: "rem", Args: []Term{Integer(-21), Integer(4)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: "rem", Args: []Term{Integer(-21), Float(4)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(4)), err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: "rem", Args: []Term{Float(-21), Integer(4)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(-21)), err)
		assert.False(t, ok)
	})

	t.Run("mod", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Integer(3), &Compound{Functor: "mod", Args: []Term{Integer(-21), Integer(4)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: "mod", Args: []Term{Integer(-21), Float(4)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(4)), err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: "mod", Args: []Term{Float(-21), Integer(4)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(-21)), err)
		assert.False(t, ok)
	})

	t.Run("exponential", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Float(16), &Compound{Functor: "**", Args: []Term{Integer(4), Integer(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(16), &Compound{Functor: "**", Args: []Term{Integer(4), Float(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(16), &Compound{Functor: "**", Args: []Term{Float(4), Integer(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(16), &Compound{Functor: "**", Args: []Term{Float(4), Float(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("sign reversal", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Integer(-2), &Compound{Functor: "-", Args: []Term{Integer(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(-2), &Compound{Functor: "-", Args: []Term{Float(2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("absolute value", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Float(2), &Compound{Functor: "abs", Args: []Term{Integer(-2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(2), &Compound{Functor: "abs", Args: []Term{Float(-2)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("arctangent", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Float(0), &Compound{Functor: "atan", Args: []Term{Integer(0)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(0), &Compound{Functor: "atan", Args: []Term{Float(0)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("ceiling", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Float(1), &Compound{Functor: "ceiling", Args: []Term{Integer(1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(1), &Compound{Functor: "ceiling", Args: []Term{Float(0.9)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("cosine", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Float(1.0), &Compound{Functor: "cos", Args: []Term{Integer(0)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(1.0), &Compound{Functor: "cos", Args: []Term{Float(0)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("natural antilogarithm", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Float(1.0), &Compound{Functor: "exp", Args: []Term{Integer(0)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(1.0), &Compound{Functor: "exp", Args: []Term{Float(0)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("square root", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Float(1.0), &Compound{Functor: "sqrt", Args: []Term{Integer(1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(1.0), &Compound{Functor: "sqrt", Args: []Term{Float(1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("sign", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Integer(1), &Compound{Functor: "sign", Args: []Term{Integer(1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Integer(1), &Compound{Functor: "sign", Args: []Term{Integer(math.MaxInt64)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Integer(0), &Compound{Functor: "sign", Args: []Term{Integer(0)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Integer(-1), &Compound{Functor: "sign", Args: []Term{Integer(-1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Integer(-1), &Compound{Functor: "sign", Args: []Term{Integer(math.MinInt64)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(1), &Compound{Functor: "sign", Args: []Term{Float(1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(1), &Compound{Functor: "sign", Args: []Term{Float(math.MaxFloat64)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(0), &Compound{Functor: "sign", Args: []Term{Float(0)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(-1), &Compound{Functor: "sign", Args: []Term{Float(-1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(-1), &Compound{Functor: "sign", Args: []Term{Float(-math.MaxFloat64)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		v := Variable("N")
		ok, err = DefaultFunctionSet.Is(v, &Compound{Functor: "sign", Args: []Term{Float(math.NaN())}}, func(env Env) Promise {
			assert.True(t, math.IsNaN(float64(env.Resolve(v).(Float))))
			return Bool(true)
		}, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("float", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Float(1.0), &Compound{Functor: "float", Args: []Term{Integer(1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(1.0), &Compound{Functor: "float", Args: []Term{Float(1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("floor", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Float(1), &Compound{Functor: "floor", Args: []Term{Integer(1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(1), &Compound{Functor: "floor", Args: []Term{Float(1.1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("natural logarithm", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Float(0), &Compound{Functor: "log", Args: []Term{Integer(1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(0), &Compound{Functor: "log", Args: []Term{Float(1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("sine", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Float(0), &Compound{Functor: "sin", Args: []Term{Integer(0)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(0), &Compound{Functor: "sin", Args: []Term{Float(0)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("truncate", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Float(1), &Compound{Functor: "truncate", Args: []Term{Integer(1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(1), &Compound{Functor: "truncate", Args: []Term{Float(1.1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("round", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Float(1), &Compound{Functor: "round", Args: []Term{Integer(1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(Float(1), &Compound{Functor: "round", Args: []Term{Float(1.1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("bit-shift right", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Integer(2), &Compound{Functor: ">>", Args: []Term{Integer(4), Integer(1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: ">>", Args: []Term{Float(4), Integer(1)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(4)), err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: ">>", Args: []Term{Integer(4), Float(1)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(1)), err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: ">>", Args: []Term{Float(4), Float(1)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(4)), err)
		assert.False(t, ok)
	})

	t.Run("bit-shift left", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Integer(8), &Compound{Functor: "<<", Args: []Term{Integer(4), Integer(1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: "<<", Args: []Term{Float(4), Integer(1)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(4)), err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: "<<", Args: []Term{Integer(4), Float(1)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(1)), err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: "<<", Args: []Term{Float(4), Float(1)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(4)), err)
		assert.False(t, ok)
	})

	t.Run("bitwise and", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Integer(1), &Compound{Functor: "/\\", Args: []Term{Integer(5), Integer(1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: "/\\", Args: []Term{Float(5), Integer(1)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(5)), err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: "/\\", Args: []Term{Integer(5), Float(1)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(1)), err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: "/\\", Args: []Term{Float(5), Float(1)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(5)), err)
		assert.False(t, ok)
	})

	t.Run("bitwise or", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Integer(5), &Compound{Functor: "\\/", Args: []Term{Integer(4), Integer(1)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: "\\/", Args: []Term{Float(4), Integer(1)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(4)), err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: "\\/", Args: []Term{Integer(4), Float(1)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(1)), err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: "\\/", Args: []Term{Float(4), Float(1)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(4)), err)
		assert.False(t, ok)
	})

	t.Run("bitwise complement", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Is(Integer(-1), &Compound{Functor: "\\", Args: []Term{Integer(0)}}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Is(NewVariable(), &Compound{Functor: "\\", Args: []Term{Float(0)}}, Success, &env).Force()
		assert.Equal(t, typeErrorInteger(Float(0)), err)
		assert.False(t, ok)
	})

	t.Run("expression is a variable", func(t *testing.T) {
		env := Env{}
		expression := Variable("Exp")

		ok, err := DefaultFunctionSet.Is(Integer(0), expression, Success, &env).Force()
		assert.Equal(t, instantiationError(expression), err)
		assert.False(t, ok)
	})
}

func TestFunctionSet_Equal(t *testing.T) {
	t.Run("same", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Equal(Integer(1), Integer(1), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Equal(Float(1), Integer(1), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Equal(Integer(1), Float(1), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.Equal(Float(1), Float(1), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("different", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.Equal(Integer(1), Integer(2), Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.Equal(Float(1), Integer(2), Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.Equal(Integer(1), Float(2), Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.Equal(Float(1), Float(2), Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("lhs is a variable", func(t *testing.T) {
		env := Env{}
		lhs := Variable("LHS")

		ok, err := DefaultFunctionSet.Equal(lhs, Integer(1), Success, &env).Force()
		assert.Equal(t, instantiationError(lhs), err)
		assert.False(t, ok)
	})

	t.Run("rhs is a variable", func(t *testing.T) {
		env := Env{}
		rhs := Variable("RHS")

		ok, err := DefaultFunctionSet.Equal(Integer(1), rhs, Success, &env).Force()
		assert.Equal(t, instantiationError(rhs), err)
		assert.False(t, ok)
	})
}

func TestFunctionSet_NotEqual(t *testing.T) {
	t.Run("same", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.NotEqual(Integer(1), Integer(1), Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.NotEqual(Float(1), Integer(1), Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.NotEqual(Integer(1), Float(1), Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)

		ok, err = DefaultFunctionSet.NotEqual(Float(1), Float(1), Success, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("different", func(t *testing.T) {
		env := Env{}
		ok, err := DefaultFunctionSet.NotEqual(Integer(1), Integer(2), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.NotEqual(Float(1), Integer(2), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.NotEqual(Integer(1), Float(2), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = DefaultFunctionSet.NotEqual(Float(1), Float(2), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("lhs is a variable", func(t *testing.T) {
		env := Env{}
		lhs := Variable("LHS")

		ok, err := DefaultFunctionSet.NotEqual(lhs, Integer(1), Success, &env).Force()
		assert.Equal(t, instantiationError(lhs), err)
		assert.False(t, ok)
	})

	t.Run("rhs is a variable", func(t *testing.T) {
		env := Env{}
		rhs := Variable("RHS")

		ok, err := DefaultFunctionSet.NotEqual(Integer(1), rhs, Success, &env).Force()
		assert.Equal(t, instantiationError(rhs), err)
		assert.False(t, ok)
	})
}

func TestFunctionSet_LessThan(t *testing.T) {
	env := Env{}
	ok, err := DefaultFunctionSet.LessThan(Integer(1), Integer(2), Success, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = DefaultFunctionSet.LessThan(Float(1), Integer(2), Success, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = DefaultFunctionSet.LessThan(Integer(1), Float(2), Success, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = DefaultFunctionSet.LessThan(Float(1), Float(2), Success, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestFunctionSet_GreaterThan(t *testing.T) {
	env := Env{}
	ok, err := DefaultFunctionSet.GreaterThan(Integer(2), Integer(1), Success, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = DefaultFunctionSet.GreaterThan(Float(2), Integer(1), Success, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = DefaultFunctionSet.GreaterThan(Integer(2), Float(1), Success, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = DefaultFunctionSet.GreaterThan(Float(2), Float(1), Success, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestFunctionSet_LessThanOrEqual(t *testing.T) {
	env := Env{}
	ok, err := DefaultFunctionSet.LessThanOrEqual(Integer(1), Integer(2), Success, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = DefaultFunctionSet.LessThanOrEqual(Float(1), Integer(2), Success, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = DefaultFunctionSet.LessThanOrEqual(Integer(1), Float(2), Success, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = DefaultFunctionSet.LessThanOrEqual(Float(1), Float(2), Success, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestFunctionSet_GreaterThanOrEqual(t *testing.T) {
	env := Env{}
	ok, err := DefaultFunctionSet.GreaterThanOrEqual(Integer(2), Integer(1), Success, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = DefaultFunctionSet.GreaterThanOrEqual(Float(2), Integer(1), Success, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = DefaultFunctionSet.GreaterThanOrEqual(Integer(2), Float(1), Success, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = DefaultFunctionSet.GreaterThanOrEqual(Float(2), Float(1), Success, &env).Force()
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestVM_StreamProperty(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	assert.NoError(t, err)

	defer func() {
		assert.NoError(t, os.Remove(f.Name()))
	}()

	t.Run("stream", func(t *testing.T) {
		env := Env{}
		expected := []Term{
			&Compound{Functor: "mode", Args: []Term{Atom("read")}},
			&Compound{Functor: "alias", Args: []Term{Atom("null")}},
			&Compound{Functor: "eof_action", Args: []Term{Atom("eof_code")}},
			Atom("input"),
			&Compound{Functor: "buffer", Args: []Term{Atom("true")}},
			&Compound{Functor: "file_name", Args: []Term{Atom(f.Name())}},
			&Compound{Functor: "position", Args: []Term{Integer(0)}},
			&Compound{Functor: "end_of_stream", Args: []Term{Atom("at")}},
			&Compound{Functor: "reposition", Args: []Term{Atom("false")}},
			&Compound{Functor: "type", Args: []Term{Atom("text")}},
		}

		v := Variable("V")
		c := 0
		var vm VM
		ok, err := vm.StreamProperty(&Stream{
			source: bufio.NewReader(f),
			closer: f,
			mode:   streamModeRead,
			alias:  "null",
		}, v, func(env Env) Promise {
			assert.Equal(t, expected[c], env.Resolve(v))
			c++
			return Bool(false)
		}, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("stream alias", func(t *testing.T) {
		env := Env{}
		expected := []Term{
			&Compound{Functor: "mode", Args: []Term{Atom("write")}},
			&Compound{Functor: "alias", Args: []Term{Atom("null")}},
			&Compound{Functor: "eof_action", Args: []Term{Atom("eof_code")}},
			Atom("output"),
			&Compound{Functor: "buffer", Args: []Term{Atom("true")}},
			&Compound{Functor: "file_name", Args: []Term{Atom(f.Name())}},
			&Compound{Functor: "position", Args: []Term{Integer(0)}},
			&Compound{Functor: "end_of_stream", Args: []Term{Atom("at")}},
			&Compound{Functor: "reposition", Args: []Term{Atom("false")}},
			&Compound{Functor: "type", Args: []Term{Atom("text")}},
		}

		vm := VM{
			streams: map[Term]*Stream{
				Atom("null"): {
					sink:   bufio.NewWriter(f),
					closer: f,
					mode:   streamModeWrite,
					alias:  "null",
				},
			},
		}
		v := Variable("V")
		c := 0
		ok, err := vm.StreamProperty(Atom("null"), v, func(env Env) Promise {
			assert.Equal(t, expected[c], env.Resolve(v))
			c++
			return Bool(false)
		}, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("correct property value", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.StreamProperty(&Stream{mode: streamModeRead}, &Compound{
			Functor: "mode",
			Args:    []Term{Atom("read")},
		}, Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("streamOrAlias is neither a variable, a stream-term, nor an alias", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.StreamProperty(Integer(0), NewVariable(), Success, &env).Force()
		assert.Equal(t, domainErrorStreamOrAlias(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("property is neither a variable nor a stream property", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.StreamProperty(NewVariable(), Atom("property"), Success, &env).Force()
		assert.Equal(t, domainErrorStreamProperty(Atom("property")), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is not associated with an open stream", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.StreamProperty(Atom("foo"), NewVariable(), Success, &env).Force()
		assert.Equal(t, existenceErrorStream(Atom("foo")), err)
		assert.False(t, ok)
	})
}

func TestVM_SetStreamPosition(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	assert.NoError(t, err)

	defer func() {
		assert.NoError(t, os.Remove(f.Name()))
	}()

	t.Run("ok", func(t *testing.T) {
		env := Env{}
		s := Stream{
			source: f,
			closer: f,
			mode:   streamModeRead,
		}

		var vm VM
		ok, err := vm.SetStreamPosition(&s, Integer(0), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("streamOrAlias is a variable", func(t *testing.T) {
		env := Env{}
		streamOrAlias := Variable("Stream")

		var vm VM
		ok, err := vm.SetStreamPosition(streamOrAlias, Integer(0), Success, &env).Force()
		assert.Equal(t, instantiationError(streamOrAlias), err)
		assert.False(t, ok)
	})

	t.Run("position is a variable", func(t *testing.T) {
		env := Env{}
		s := Stream{
			source: f,
			closer: f,
			mode:   streamModeRead,
		}
		position := Variable("Pos")

		var vm VM
		ok, err := vm.SetStreamPosition(&s, position, Success, &env).Force()
		assert.Equal(t, instantiationError(position), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is neither a variable nor a stream term or alias", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.SetStreamPosition(Integer(2), Integer(0), Success, &env).Force()
		assert.Equal(t, domainErrorStreamOrAlias(Integer(2)), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias is not associated with an open stream", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.SetStreamPosition(Atom("foo"), Integer(0), Success, &env).Force()
		assert.Equal(t, existenceErrorStream(Atom("foo")), err)
		assert.False(t, ok)
	})

	t.Run("streamOrAlias has stream property reposition(false)", func(t *testing.T) {
		s := Variable("Stream")
		env := Env{
			{
				Variable: s,
				Value: &Stream{
					source: bytes.NewReader(nil),
					mode:   streamModeRead,
				},
			},
		}

		var vm VM
		ok, err := vm.SetStreamPosition(s, Integer(0), Success, &env).Force()
		assert.Equal(t, permissionError(Atom("reposition"), Atom("stream"), s, Atom("Stream is not a file.")), err)
		assert.False(t, ok)
	})
}

func TestVM_CharConversion(t *testing.T) {
	t.Run("register", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.CharConversion(Atom("a"), Atom("b"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		assert.Equal(t, 'b', vm.charConversions['a'])
	})

	t.Run("remove", func(t *testing.T) {
		env := Env{}
		vm := VM{
			charConversions: map[rune]rune{
				'a': 'b',
			},
		}
		ok, err := vm.CharConversion(Atom("a"), Atom("a"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		_, ok = vm.charConversions['a']
		assert.False(t, ok)
	})

	t.Run("inChar is a variable", func(t *testing.T) {
		env := Env{}
		inChar := Variable("In")

		var vm VM
		ok, err := vm.CharConversion(inChar, Atom("a"), Success, &env).Force()
		assert.Equal(t, instantiationError(inChar), err)
		assert.False(t, ok)
	})

	t.Run("outChar is a variable", func(t *testing.T) {
		env := Env{}
		outChar := Variable("Out")

		var vm VM
		ok, err := vm.CharConversion(Atom("a"), outChar, Success, &env).Force()
		assert.Equal(t, instantiationError(outChar), err)
		assert.False(t, ok)
	})

	t.Run("inChar is neither a variable nor a one character atom", func(t *testing.T) {
		t.Run("not even an atom", func(t *testing.T) {
			env := Env{}
			var vm VM
			ok, err := vm.CharConversion(Integer(0), Atom("a"), Success, &env).Force()
			assert.Equal(t, representationError(Atom("character"), Atom("0 is not a character.")), err)
			assert.False(t, ok)
		})

		t.Run("multi-character atom", func(t *testing.T) {
			env := Env{}
			var vm VM
			ok, err := vm.CharConversion(Atom("foo"), Atom("a"), Success, &env).Force()
			assert.Equal(t, representationError(Atom("character"), Atom("foo is not a character.")), err)
			assert.False(t, ok)
		})
	})

	t.Run("outChar is neither a variable nor a one character atom", func(t *testing.T) {
		t.Run("not even an atom", func(t *testing.T) {
			env := Env{}
			var vm VM
			ok, err := vm.CharConversion(Atom("a"), Integer(0), Success, &env).Force()
			assert.Equal(t, representationError(Atom("character"), Atom("0 is not a character.")), err)
			assert.False(t, ok)
		})

		t.Run("multi-character atom", func(t *testing.T) {
			env := Env{}
			var vm VM
			ok, err := vm.CharConversion(Atom("a"), Atom("foo"), Success, &env).Force()
			assert.Equal(t, representationError(Atom("character"), Atom("foo is not a character.")), err)
			assert.False(t, ok)
		})
	})
}

func TestVM_CurrentCharConversion(t *testing.T) {
	t.Run("specified", func(t *testing.T) {
		t.Run("as is", func(t *testing.T) {
			env := Env{}
			var vm VM
			ok, err := vm.CurrentCharConversion(Atom("a"), Atom("a"), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})

		t.Run("converted", func(t *testing.T) {
			env := Env{}
			vm := VM{
				charConversions: map[rune]rune{
					'a': 'b',
				},
			}
			ok, err := vm.CurrentCharConversion(Atom("a"), Atom("b"), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
		})
	})

	t.Run("not specified", func(t *testing.T) {
		env := Env{}
		x, y := Variable("X"), Variable("Y")

		var r rune
		var vm VM
		ok, err := vm.CurrentCharConversion(x, y, func(env Env) Promise {
			ref, ok := env.Lookup(x)
			assert.True(t, ok)
			x, ok := ref.(Atom)
			assert.True(t, ok)
			assert.Len(t, []rune(x), 1)

			ref, ok = env.Lookup(y)
			assert.True(t, ok)
			y, ok := ref.(Atom)
			assert.True(t, ok)
			assert.Len(t, []rune(y), 1)

			assert.Equal(t, r, []rune(x)[0])
			assert.Equal(t, r, []rune(y)[0])
			r++
			return Bool(false)
		}, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, rune(256), r)
	})

	t.Run("inChar is neither a variable nor a one character atom", func(t *testing.T) {
		t.Run("not even an atom", func(t *testing.T) {
			env := Env{}
			var vm VM
			ok, err := vm.CurrentCharConversion(Integer(0), Atom("b"), Success, &env).Force()
			assert.Equal(t, representationError(Atom("character"), Atom("0 is not a character.")), err)
			assert.False(t, ok)
		})

		t.Run("multi-character atom", func(t *testing.T) {
			env := Env{}
			var vm VM
			ok, err := vm.CurrentCharConversion(Atom("foo"), Atom("b"), Success, &env).Force()
			assert.Equal(t, representationError(Atom("character"), Atom("foo is not a character.")), err)
			assert.False(t, ok)
		})
	})

	t.Run("outChar is neither a variable nor a one character atom", func(t *testing.T) {
		t.Run("not even an atom", func(t *testing.T) {
			env := Env{}
			var vm VM
			ok, err := vm.CurrentCharConversion(Atom("a"), Integer(0), Success, &env).Force()
			assert.Equal(t, representationError(Atom("character"), Atom("0 is not a character.")), err)
			assert.False(t, ok)
		})

		t.Run("multi-character atom", func(t *testing.T) {
			env := Env{}
			var vm VM
			ok, err := vm.CurrentCharConversion(Atom("a"), Atom("bar"), Success, &env).Force()
			assert.Equal(t, representationError(Atom("character"), Atom("bar is not a character.")), err)
			assert.False(t, ok)
		})
	})
}

func TestVM_SetPrologFlag(t *testing.T) {
	t.Run("bounded", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.SetPrologFlag(Atom("bounded"), NewVariable(), Success, &env).Force()
		assert.Equal(t, permissionError(Atom("modify"), Atom("flag"), Atom("bounded"), Atom("bounded is not modifiable.")), err)
		assert.False(t, ok)
	})

	t.Run("max_integer", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.SetPrologFlag(Atom("max_integer"), NewVariable(), Success, &env).Force()
		assert.Equal(t, permissionError(Atom("modify"), Atom("flag"), Atom("max_integer"), Atom("max_integer is not modifiable.")), err)
		assert.False(t, ok)
	})

	t.Run("min_integer", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.SetPrologFlag(Atom("min_integer"), NewVariable(), Success, &env).Force()
		assert.Equal(t, permissionError(Atom("modify"), Atom("flag"), Atom("min_integer"), Atom("min_integer is not modifiable.")), err)
		assert.False(t, ok)
	})

	t.Run("integer_rounding_function", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.SetPrologFlag(Atom("integer_rounding_function"), NewVariable(), Success, &env).Force()
		assert.Equal(t, permissionError(Atom("modify"), Atom("flag"), Atom("integer_rounding_function"), Atom("integer_rounding_function is not modifiable.")), err)
		assert.False(t, ok)
	})

	t.Run("char_conversion", func(t *testing.T) {
		t.Run("on", func(t *testing.T) {
			env := Env{}
			var vm VM
			ok, err := vm.SetPrologFlag(Atom("char_conversion"), Atom("on"), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.True(t, vm.charConvEnabled)
		})

		t.Run("off", func(t *testing.T) {
			env := Env{}
			vm := VM{charConvEnabled: true}
			ok, err := vm.SetPrologFlag(Atom("char_conversion"), Atom("off"), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.False(t, vm.charConvEnabled)
		})
	})

	t.Run("debug", func(t *testing.T) {
		t.Run("on", func(t *testing.T) {
			env := Env{}
			var vm VM
			ok, err := vm.SetPrologFlag(Atom("debug"), Atom("on"), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.True(t, vm.debug)
		})

		t.Run("off", func(t *testing.T) {
			env := Env{}
			vm := VM{debug: true}
			ok, err := vm.SetPrologFlag(Atom("debug"), Atom("off"), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.False(t, vm.debug)
		})
	})

	t.Run("max_arity", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.SetPrologFlag(Atom("max_arity"), NewVariable(), Success, &env).Force()
		assert.Equal(t, permissionError(Atom("modify"), Atom("flag"), Atom("max_arity"), Atom("max_arity is not modifiable.")), err)
		assert.False(t, ok)
	})

	t.Run("unknown", func(t *testing.T) {
		t.Run("error", func(t *testing.T) {
			env := Env{}
			vm := VM{unknown: unknownFail}
			ok, err := vm.SetPrologFlag(Atom("unknown"), Atom("error"), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.Equal(t, unknownError, vm.unknown)
		})

		t.Run("warning", func(t *testing.T) {
			env := Env{}
			var vm VM
			ok, err := vm.SetPrologFlag(Atom("unknown"), Atom("warning"), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.Equal(t, unknownWarning, vm.unknown)
		})

		t.Run("fail", func(t *testing.T) {
			env := Env{}
			var vm VM
			ok, err := vm.SetPrologFlag(Atom("unknown"), Atom("fail"), Success, &env).Force()
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.Equal(t, unknownFail, vm.unknown)
		})
	})

	t.Run("flag is a variable", func(t *testing.T) {
		env := Env{}
		flag := Variable("Flag")

		var vm VM
		ok, err := vm.SetPrologFlag(flag, Atom("fail"), Success, &env).Force()
		assert.Equal(t, instantiationError(flag), err)
		assert.False(t, ok)
	})

	t.Run("value is a variable", func(t *testing.T) {
		env := Env{}
		value := Variable("Value")

		var vm VM
		ok, err := vm.SetPrologFlag(Atom("unknown"), value, Success, &env).Force()
		assert.Equal(t, instantiationError(value), err)
		assert.False(t, ok)
	})

	t.Run("flag is neither a variable nor an atom", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.SetPrologFlag(Integer(0), Atom("fail"), Success, &env).Force()
		assert.Equal(t, typeErrorAtom(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("flag is an atom but an invalid flag for the processor", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.SetPrologFlag(Atom("foo"), Atom("fail"), Success, &env).Force()
		assert.Equal(t, domainErrorPrologFlag(Atom("foo")), err)
		assert.False(t, ok)
	})

	t.Run("value is inadmissible for flag", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.SetPrologFlag(Atom("unknown"), Integer(0), Success, &env).Force()
		assert.Equal(t, domainErrorFlagValue(&Compound{
			Functor: "+",
			Args:    []Term{Atom("unknown"), Integer(0)},
		}), err)
		assert.False(t, ok)
	})

	t.Run("value is admissible for flag but the flag is not modifiable", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.SetPrologFlag(Atom("bounded"), Atom("true"), Success, &env).Force()
		assert.Equal(t, permissionError(Atom("modify"), Atom("flag"), Atom("bounded"), Atom("bounded is not modifiable.")), err)
		assert.False(t, ok)
	})
}

func TestVM_CurrentPrologFlag(t *testing.T) {
	var vm VM

	t.Run("specified", func(t *testing.T) {
		env := Env{}
		ok, err := vm.CurrentPrologFlag(Atom("bounded"), Atom("true"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = vm.CurrentPrologFlag(Atom("max_integer"), Integer(math.MaxInt64), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = vm.CurrentPrologFlag(Atom("min_integer"), Integer(math.MinInt64), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = vm.CurrentPrologFlag(Atom("integer_rounding_function"), Atom("toward_zero"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = vm.CurrentPrologFlag(Atom("char_conversion"), Atom("off"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = vm.CurrentPrologFlag(Atom("debug"), Atom("off"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = vm.CurrentPrologFlag(Atom("max_arity"), Atom("unbounded"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = vm.CurrentPrologFlag(Atom("unknown"), Atom("error"), Success, &env).Force()
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("not specified", func(t *testing.T) {
		env := Env{}
		flag, value := Variable("Flag"), Variable("Value")
		var c int
		ok, err := vm.CurrentPrologFlag(flag, value, func(env Env) Promise {
			switch c {
			case 0:
				assert.Equal(t, Atom("bounded"), env.Resolve(flag))
				assert.Equal(t, Atom("true"), env.Resolve(value))
			case 1:
				assert.Equal(t, Atom("max_integer"), env.Resolve(flag))
				assert.Equal(t, Integer(math.MaxInt64), env.Resolve(value))
			case 2:
				assert.Equal(t, Atom("min_integer"), env.Resolve(flag))
				assert.Equal(t, Integer(math.MinInt64), env.Resolve(value))
			case 3:
				assert.Equal(t, Atom("integer_rounding_function"), env.Resolve(flag))
				assert.Equal(t, Atom("toward_zero"), env.Resolve(value))
			case 4:
				assert.Equal(t, Atom("char_conversion"), env.Resolve(flag))
				assert.Equal(t, Atom("off"), env.Resolve(value))
			case 5:
				assert.Equal(t, Atom("debug"), env.Resolve(flag))
				assert.Equal(t, Atom("off"), env.Resolve(value))
			case 6:
				assert.Equal(t, Atom("max_arity"), env.Resolve(flag))
				assert.Equal(t, Atom("unbounded"), env.Resolve(value))
			case 7:
				assert.Equal(t, Atom("unknown"), env.Resolve(flag))
				assert.Equal(t, Atom(vm.unknown.String()), env.Resolve(value))
			default:
				assert.Fail(t, "unreachable")
			}
			c++
			return Bool(false)
		}, &env).Force()
		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, 8, c)
	})

	t.Run("flag is neither a variable nor an atom", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.CurrentPrologFlag(Integer(0), Atom("error"), Success, &env).Force()
		assert.Equal(t, typeErrorAtom(Integer(0)), err)
		assert.False(t, ok)
	})

	t.Run("flag is an atom but an invalid flag for the processor", func(t *testing.T) {
		env := Env{}
		var vm VM
		ok, err := vm.CurrentPrologFlag(Atom("foo"), Atom("error"), Success, &env).Force()
		assert.Equal(t, domainErrorPrologFlag(Atom("foo")), err)
		assert.False(t, ok)
	})
}
