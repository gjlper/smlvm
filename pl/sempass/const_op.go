package sempass

import (
	"shanhu.io/smlvm/lexing"
	"shanhu.io/smlvm/pl/tast"
	"shanhu.io/smlvm/pl/types"
)

func unaryOpConst(b *builder, opTok *lexing.Token, B tast.Expr) tast.Expr {
	op := opTok.Lit
	bref := B.R()
	if !bref.IsSingle() {
		b.CodeErrorf(opTok.Pos, "pl.invalidExprStmt",
			"invalid expression, not single: %q on %s", op, bref)
		return nil
	}

	v, ok := types.NumConst(bref.T)
	if ok {
		switch op {
		case "+":
			return B // just shortcut this
		case "-":
			return tast.NewConst(tast.NewRef(types.NewNumber(-v)))
		}
		b.CodeErrorf(opTok.Pos, "pl.invalidOp",
			"invalid operation on num const: %q on %s", op, B)
		return nil
	}

	ct, ok := bref.T.(*types.Const)
	if !ok {
		b.CodeErrorf(opTok.Pos, "pl.expectConstExpr",
			"expect const expression but got %q %s", op, B)
		return nil
	}
	t := ct.Type
	if types.IsBasic(t, types.Bool) {
		// TODO
		b.CodeErrorf(opTok.Pos, "pl.notYetSupported",
			"const bool is not supported yet")
		return nil
	}
	if types.IsBasic(t, types.Float32) {
		b.CodeErrorf(opTok.Pos, "pl.notYetSupported",
			"float is not supported yet")
		return nil
	}
	if types.IsInteger(t) {
		switch op {
		case "+":
			return B // just shortcut this
		case "-":
			return tast.NewConst(tast.NewRef(types.NewConstInt(-v, t)))
		}
		b.CodeErrorf(opTok.Pos, "pl.invalidOp",
			"invalid operation on int const: %q on %s", op, B)
		return nil
	}

	b.CodeErrorf(opTok.Pos, "pl.impossible",
		"only basic type const supported")
	return nil
}

func binaryOpConst(b *builder, opTok *lexing.Token, A, B tast.Expr) tast.Expr {
	op := opTok.Lit
	aref := A.R()
	bref := B.R()
	if !(aref.IsSingle() && bref.IsSingle()) {
		b.CodeErrorf(opTok.Pos, "pl.notSingle",
			"only single expr supported for const %s %q %s", aref.T, op, bref.T)
		return nil
	}
	ca, oka := aref.Type().(*types.Const)
	cb, okb := bref.Type().(*types.Const)
	if !(oka && okb) {
		b.CodeErrorf(opTok.Pos, "pl.expectConstExpr",
			"expect a const expression, got %s %q %s", aref.T, op, bref.T)
		return nil
	}

	va, oka := types.NumConst(aref.Type())
	vb, okb := types.NumConst(bref.Type())
	if oka && okb {
		return binaryOpNums(b, opTok, A, B)
	}

	if oka || okb {
		if oka && types.IsInteger(cb.Type) {
			_ = va
		}
		if okb && types.IsInteger(ca.Type) {
			_ = vb
		}
	}
	b.CodeErrorf(opTok.Pos, "pl.notYetSupported",
		"only num and int consts are implemented")
	return nil
}

// TODO: after added const bool, change input into va, ab
func binaryOpNums(b *builder, opTok *lexing.Token, A, B tast.Expr) tast.Expr {
	op := opTok.Lit
	va, _ := types.NumConst(A.R().Type())
	vb, _ := types.NumConst(B.R().Type())
	r := func(v int64) tast.Expr {
		return tast.NewConst(tast.NewRef(types.NewNumber(v)))
	}

	switch op {
	case "+":
		return r(va + vb)
	case "-":
		return r(va - vb)
	case "*":
		return r(va * vb)
	case "&":
		return r(va & vb)
	case "|":
		return r(va | vb)
	case "^":
		return r(va ^ vb)
	case "%":
		if vb == 0 {
			b.CodeErrorf(opTok.Pos, "pl.divideByZero", "modular by zero")
			return nil
		}
		return r(va % vb)
	case "/":
		if vb == 0 {
			b.CodeErrorf(opTok.Pos, "pl.divideByZero", "divide by zero")
			return nil
		}
		return r(va / vb)
	case "==", "!=", ">", "<", ">=", "<=":
		// TODO: will change into a const bool
		return &tast.OpExpr{
			A: A, Op: opTok, B: B,
			Ref: tast.NewRef(types.Bool),
		}
	case "<<":
		if vb < 0 {
			b.Errorf(opTok.Pos, "shift with negative value: %d", vb)
			return nil
		}
		return r(va << uint64(vb))
	case ">>":
		if vb < 0 {
			b.Errorf(opTok.Pos, "shift with negative value: %d", vb)
			return nil
		}
		return r(va >> uint64(vb))
	}

	b.CodeErrorf(opTok.Pos, "pl.invalidOp", "%q on num consts", op)
	return nil
}
