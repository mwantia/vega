# Vega Statements

```
name         = Statement keywords
|            = Separates syntax items enclosed in brackets or braces. You can use only one of the items.
[]           = Optional syntax item(s).
{}           = Required syntax items.
[,...n]      = Indicates the preceding item can be repeated n number of times. The occurrences are separated by ','.
[...n]       = Indicates the preceding item can be repeated n number of times. The occurrences are separated by ' '.
;            = Statement terminator (not required). 
<label> ::   = The name for a block of syntax
```
```
var                 value   : string | integer = "Foo Bar" 
      let           value   : integer          = 2015
            const   value   : boolean | null   = true 
var                 value                      = "Hello World" 

type hello   = string | null 
type world   = "Foo" | "Bar" | boolean 
```

## Known Types

| Name    | Allocable | Numeric | Viewable | Description |
|---------|-----------|---------|----------|-------------|
| boolean | Yes       | No      | No       |             |
| byte    | Yes       | Yes     | No       |             |
| short   | Yes       | Yes     | No       |             |
| integer | Yes       | Yes     | No       |             |
| long    | Yes       | Yes     | No       |             |
| float   | Yes       | Yes     | No       |             |
| string  | No        | No      | Yes      |             |
| slice   | No        | No      | Yes      |             |
| result  | ?         | No      | No       |             |
| handler | No        | No      | Yes      |             |
| meta    | Yes       | No      | No       |             |