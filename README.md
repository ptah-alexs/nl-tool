# nl-tool
ii/idec nodelist manipulation tool

## build

```
git clone https://github.com/ptah-alexs/nl-tool
cd nl-tool
go get github.com/ptah-alexs/nl-tool
go build -ldflags "-s -w"
```
## help
```
$ ./nl-tool 
Help: ./nl-tool [options] command [arguments]
Commands:
        check                         - check availability station from nodelist
        export [file]                 - export nodes to file (default: nodes.txt)
        import [file]                 - import nodes from file (default: nodes.txt)
        info                          - print actual nodelist
        sync                          - sync nodelists from other station

Options:
        -db=<path>                    - database path (default: station.txt)
        -a                            - check, sync: process all entries, without excludes
        -i                            - sync: ignore timestamp for altpath stations
        -s                            - sync: skip check ii-compatibility for entries
```

check  - проверяет станции из своего списка на доступность, если нет корректного ответа в течение недели, то станция маскируется и не участвует больше в проверке и не экспортируется в nodes.txt
sync   - проверяет станции из своего списка на наличие nodes.txt и берёт из них незнакомые станции к себе. Станции в базе отмеченные признаком exclude не участвуют в синхронизации.
info   - выводит список всех станций из своей базы.
export - формирует nodes.txt из своей базы, замаскированные станции не попадают в этот файл
import - загружает список незнакомых станций из nodes.txt

-a включает в проверку замаскированные станции для check и станции с признаком exclude для sync
-i отключает проверку по времени последнего доступа к станциям с признаком altpath
-s отключает проверку на корректный ответ ii-станции для sync

## Коротко об altpath
Так как ii/idec распределенная сеть, то часть станций может быть недоступна напрямую, поэтому в nodes.txt есть признак altpath который означает что этот сервер недоступен для этой конкретной станции, но есть в списке нод других станций, т.е. где-то доступен. Для altpath-станций также проверяется время последней удачной проверки, т.е. если время проверки в nodes.txt откуда взята станция больше недели то она маскируется. Такое поведение отключается опцией -i.

## Формат nodes.txt
Файл содержит поля url, имя, время последней удачной проверки в unixtime, признак altpath (+ для истинного значения, - для ложного), разделённые табом. Обязательно только поле url. Пример файла:
```
http://localhost:8080/        local        1732651250        +
http://example.com/        local        1732651250        -
```
## Формат бд
Файл содержит поля url, имя, время последней удачной проверки в unixtime, признак маскировки, признак altpath, признак exclude. Три последних поля логические значения, где + истина, - ложь.

```
http://localhost:8080/        local        1732651250        -        +        -
http://example.com/        local        1732651250        -        -        +
```
