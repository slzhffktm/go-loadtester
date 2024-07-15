# go-loadtester

Simple load testing tool for Go. It is designed to support multistep HTTP testing and will store & calculate the HTTP metrics.

## Why did I build this tool?

Because the existing tools cannot support getting the HTTP response, do something, then use it for the next request.
This tool is specifically designed for this purpose.
Example: Create an object, then get the created object using the ID returned from the create object response.

## Installation

```bash
go get -u github.com/slzhffktm/go-loadtester
```

## How to Use

Check the example at [example/main.go](example/main.go).
It'll print this result:

```bash
Summaries
+---------------+----------------+-----------+--------------+-------------+
|     NAME      | TOTAL REQUESTS | SUCCESS % | TOTAL ERRORS | ERROR LISTS |
+---------------+----------------+-----------+--------------+-------------+
| Create object |            100 | 100.00 %  |            0 | []          |
| Get object    |            100 | 100.00 %  |            0 | []          |
+---------------+----------------+-----------+--------------+-------------+

Status Codes
+---------------+-----+
|     NAME      | 200 |
+---------------+-----+
| Create object | 100 |
| Get object    | 100 |
+---------------+-----+

Latencies
+---------------+--------+--------+--------+--------+--------+--------+-------+
|     NAME      |  50%   |  90%   |  95%   |  99%   |  AVG   |  MAX   |  MIN  |
+---------------+--------+--------+--------+--------+--------+--------+-------+
| Create object | 106 ms | 233 ms | 348 ms | 356 ms | 129 ms | 357 ms | 88 ms |
| Get object    | 86 ms  | 96 ms  | 99 ms  | 106 ms | 87 ms  | 107 ms | 75 ms |
+---------------+--------+--------+--------+--------+--------+--------+-------+

Latencies (Success Only)
+---------------+--------+--------+--------+--------+--------+--------+-------+
|     NAME      |  50%   |  90%   |  95%   |  99%   |  AVG   |  MAX   |  MIN  |
+---------------+--------+--------+--------+--------+--------+--------+-------+
| Create object | 106 ms | 233 ms | 348 ms | 356 ms | 129 ms | 357 ms | 88 ms |
| Get object    | 86 ms  | 96 ms  | 99 ms  | 106 ms | 87 ms  | 107 ms | 75 ms |
+---------------+--------+--------+--------+--------+--------+--------+-------+
```

## Thanks to

- [vegeta](https://github.com/tsenart/vegeta/tree/master)
- [autocannon-go](https://github.com/GlenTiki/autocannon-go/tree/master)
