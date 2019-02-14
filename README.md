# Telegram Photo Resize Bot

A bot that takes any photo (<4MB) that is uploaded as a document and re-uploads it as a compressed version so that a preview is shown in the telegram client

## Getting Started

These instructions will cover usage information and prerequisites for the docker container 

### Prerequisities

In order to run this container you'll need docker installed.

* [Windows](https://docs.docker.com/windows/started)
* [OS X](https://docs.docker.com/mac/started/)
* [Linux](https://docs.docker.com/linux/started/)

### Usage

#### Container Parameters

List the different parameters available to your container

```shell
docker run zackpollard/tg-photo-resize-bot -e "TG_BOT_KEY=bot_api_key_here"
```

#### Environment Variables

* `TG_BOT_KEY` - The Telegram Bot API key that will be used by this bot

## Built With

* python, dependencies can be found in requirements.txt

## Find Us

* [GitHub](https://github.com/zackpollard/tg-photo-resize-bot)
* [DockerHub](https://hub.docker.com/r/zackpollard/tg-photo-resize-bot)

## Authors

* **Zack Pollard** - *Maintainance and Conversion to Docker*

## License

This project is licensed under the Unlicense License - see the [LICENSE](LICENSE) file for details.
