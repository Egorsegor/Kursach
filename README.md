# Dr. Brain

**Сайт для создания квизов**

## Цель проекта
Создать веб-сайт для удобного создания и редактирования квизов, с возможностью анализа результатов и поддержкой различных типов вопросов.


## Паттерны
- **Singleton**: Паттерн Singleton используется для подключения к MongoDB, чтобы гарантировать наличие только одного экземпляра соединения с базой данных во всём приложении
- **Model-View-Presenter**: models/, handlers/, frontend/
- **Repository Pattern**: инкапсулированная реализация логики хранения данных в бд в config/
- **Observer**: WebSocket sessionConnections(), broadcast() функции


## Технологии
**Backend**: Go(Fiber)<br>
**Frontend**: HTML, CSS, Javascript<br>
**Database**: MongoDB<br>
**Infra**: Session Store<br>


## Быстрый старт
```bash
git clone https://github.com/Egorsegor/Dr-Brain-site-project.git
cd Dr-Brain-site-project
go mod tidy
go run main.go
