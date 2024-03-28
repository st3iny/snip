package cfg

import (
	"io/ioutil"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type Snipfile struct {
	GlobalOptions []*KeyValue `@@*`
	Backends []*Backend `("backends" LBracket EOL @@* RBracket EOL)?`
	Frontends []*Frontend `"frontends" LBracket EOL @@* RBracket EOL`
}

type Block struct {
	Options []*Option `LBracket EOL @@* RBracket`
}

type Backend struct {
	Name string `@Ident`
	Upstream string `@Ident`
	Block   *Block `@@? EOL`
}

type Frontend struct {
	Domain string `@Ident`
	Backend string `@Ident`
	Block   *Block `@@? EOL`
}

type Option struct {
	Key string `@Ident (EOL|Comment)`
}

type KeyValue struct {
	Key string `@Ident`
	Value string `@Ident (EOL|Comment)`
}

func ParseSnipfile(path string) (*Snipfile, error) {
	confLexer := lexer.MustSimple([]lexer.SimpleRule{
		{"Comment", `#[^\n]*\n`},
		//{"Addr", `[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+:[0-9]+`},
		//{"ColonPort", `:[0-9]+`},
		//{"Addr", `[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`},
		{"Ident", `[a-z0-9.-_:]+`},
		//{"String", `"(\\"|[^"])*"`},
		{"LBracket", `{`},
		{"RBracket", `}`},
		{"EOL", `[\n]+`},
		{"Whitespace", `[ \t]+`},
	})

	parser, err := participle.Build[Snipfile](
		participle.Lexer(confLexer),
		participle.Elide("Comment", "Whitespace"),
		//participle.CaseInsensitive("Domain", "Option"),
	)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return parser.ParseBytes(path, bytes)
}


func (block *Block) Has(key string) bool {
	if block == nil || block.Options == nil {
		return false
	}

	for _, opt := range block.Options {
		if opt.Key == key {
			return true
		}
	}

	return false
}

func (snip *Snipfile) HasGlobal(key string) bool {
	for _, opt := range snip.GlobalOptions {
		if opt.Key == key {
			return true
		}
	}

	return false
}

func (snip *Snipfile) GetGlobal(key string) string {
	for _, opt := range snip.GlobalOptions {
		if opt.Key == key {
			return opt.Value
		}
	}

	return ""
}
