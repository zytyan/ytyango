grammar Calc;

prog
    : stat (ENTER+ stat)* (ENTER)* EOF
    ;

stat
    : expr
    ;

expr
    : expr DICE expr       # Dice
    | <assoc=right> expr POW expr       # Pow
    | expr (MUL|DIV|MOD) expr # MulDivMod
    | expr (ADD|SUB) expr # AddSub
    | number              # Num
    | HEX                 # Hex
    | ID                  # Id
    | '(' expr ')'        # Parens
    ;
MUL : '*' ;
DIV : '/' ;
ADD : '+' ;
SUB : '-' ;
POW : '^'|'**' ;
MOD : '%' | 'mod' ;
DICE: 'd' | 'D' ;
ID  : ('pi'|'e') ;

INT : [0-9]+;
FLOAT
    : [0-9]+ '.' [0-9]* EXP?
    | '.' [0-9]+ EXP?
    | [0-9]+ EXP
    ;
EXP : [Ee] [+-]? [0-9]+ ;
HEX : '0' [xX] [0-9a-fA-F]+ ;
number: INT | FLOAT;

ENTER : '\r'?'\n' ;
WS  : [ \t]+ -> skip ;    // toss out whitespace