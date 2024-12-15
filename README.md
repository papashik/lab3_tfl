# Лаб. работа № 3 ТФЯ

### Запуск:

```
go run main.go
```

##### Флаги:

```
  -count int
    	Number of tests to generate (default 10)
  -file string
    	Output file name or 'STDOUT' (default "STDOUT")
  -format string
    	Output file format ('JSON' or 'DEFAULT') (default "DEFAULT")
  -necessary
    	If true, percentage of positive tests will be satisfied at any cost.
    	Set false if program is working too slowly or freezes (default true) (default true)
  -percent int
    	Percentage of positive tests (default 50)
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

Оба варианта завершаются строкой `END` или символом EOF.