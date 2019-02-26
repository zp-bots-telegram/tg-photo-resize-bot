FROM python:3 AS build-env
WORKDIR /bot
ADD requirements.txt .
ADD bot.py . 
RUN pip install -r requirements.txt

FROM python:3-alpine
COPY --from=build-env /bot /bot

CMD [ "python", "./bot.py" ]
