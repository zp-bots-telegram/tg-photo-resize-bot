FROM python:3 AS build-env
RUN mkdir /install
WORKDIR /bot
ADD requirements.txt .
RUN pip install --install-option="--prefix=/install" -r requirements.txt

FROM python:3-alpine
COPY --from=build-env /install /usr/local
WORKDIR /bot
ADD bot.py . 

CMD [ "python", "./bot.py" ]
