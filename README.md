# Лаб. работа № 3 ТФЯ

Генератор тестов по заданной грамматике.
### Запуск:

```
go run main.go
```

##### Флаги:

```
-count int
    Number of tests to generate (default 10)
-file string
    Output file name or "STDOUT" (default "STDOUT")
-format string
    Tests output format ("JSON" or "DEFAULT") (default "DEFAULT")
-necessary
    If true, percentage of positive tests will be satisfied at any cost.
    Set false if program is working too slowly or freezes (default false)
-percent int
    Percentage of positive tests (default 50)
-verbose
    Verbose output in STDOUT (default false)

```

### Форматы ввода грамматики:

1. По условию

```
rule := A -> B C
NT := A
NT := B
NT := C
NT := D
T := a
rule := B -> a a
rule := C -> a B a
rule := C -> D a
``` 

2. Упрощённый

```
TLIST = a
NTLIST = A B C D
A -> B C
B -> a a
C -> a B a
C -> D a
END
```

Оба варианта завершаются строкой `"END"` или символом `EOF`.