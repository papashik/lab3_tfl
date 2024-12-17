# Лаб. работа № 3 ТФЯ

Генератор тестов по заданной грамматике.
### Запуск:

```
go run main.go
```

##### Флаги:

```
-count int
    Number of tests to generate (default 20)
-format string
    Tests output format ("JSON" or "DEFAULT") (default "DEFAULT")
-input string
    Input file name or 'STDIN' (default "STDIN")
-necessary
    If set, percentage of positive tests will be satisfied at any performance cost.
    Program can freeze and work slowly while looking for positive tests
-output string
    Output file name or "STDOUT" (default "STDOUT")
-percent int
    Percentage of positive tests (default 50)
-verbose
    Verbose output in STDOUT
```

### Формат ввода грамматики:

```
TLIST = a b c
S -> S c c c S
S -> d
END
```

Завершать ввод можно строкой `"END"` или символом `EOF`.