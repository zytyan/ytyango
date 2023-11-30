grammar Calc;

prog
    : (stat (ENTER|EOF))+
    ;

stat
    : expr
    | ID '=' expr
    ;

expr
    : <assoc=right> expr POW expr       # Pow
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

ID  : [a-zA-Z]+ ;

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