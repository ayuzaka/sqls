package parseutil

import (
	"github.com/ayuzaka/sqls/ast"
	"github.com/ayuzaka/sqls/ast/astutil"
	"github.com/ayuzaka/sqls/token"
)

func ExtractIdenfiers(parsed ast.TokenList, pos token.Pos) ([]ast.Node, error) {
	stmt, err := extractFocusedStatement(parsed, pos)
	if err != nil {
		return nil, err
	}

	identifierMatcher := astutil.NodeMatcher{
		NodeTypes: []ast.NodeType{
			ast.TypeIdentifier,
		},
	}
	return parsePrefix(astutil.NewNodeReader(stmt), identifierMatcher, parseIdentifier), nil
}

func parseIdentifier(reader *astutil.NodeReader) []ast.Node {
	return []ast.Node{reader.CurNode}
}
