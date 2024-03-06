# go-musthave-diploma-tpl

## Swagger

1. Установить утилиту `swag` по инструкции https://github.com/swaggo/http-swagger
2. Сгенерировать swagger-документацию следующей командой из коревой директории:
   ```
   swag init `
       -d "./internal/entrypoints/http/v1/" `
       -g router.go `
       -o "./internal/entrypoints/http/v1/docs/"
    ```
3. Использовать `swag fmt` для форматирования аннотаций
 