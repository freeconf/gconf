module "module"
[ident] "leafref-i1"
{ "{"
revision "revision"
[string] "0"
; ";"
import "import"
[ident] "leafref-i2"
{ "{"
prefix "prefix"
[string] "two"
; ";"
} "}"
leaf "leaf"
[ident] "l1"
{ "{"
type "type"
[ident] "leafref"
{ "{"
path "path"
[string] "\"/two:l2\""
; ";"
} "}"
} "}"
} "}"
