#!/usr/bin/env python3
"""
Bot to reply to uncompressed pictures (below a certain filesize)
with compressed versions and caption of original resolution.
For https://t.me/WallpapersGroup.
"""
import logging
import tempfile
import os
from telegram.ext import Updater, MessageHandler, Filters
from PIL import Image
from fractions import Fraction
import exifread

# Enable logging
logging.basicConfig(format='%(asctime)s - %(name)s - %(levelname)s -'
                           ' %(message)s from uncompressionbot',
                    level=logging.INFO)

logger = logging.getLogger(__name__)

TYPES = ['image/jpeg', 'image/png', 'image/jpg']  # Not sure if jpg is us


def document_msg_handler(bot, update):
    msg = update.message
    if msg.document.mime_type not in TYPES:
        return
    if not msg.document.file_size:
        # We don't want to download if we don't know how big it is
        logger.info("Couldn't see filesize of image at {}".format(msg.date))
        return
    if msg.document.file_size > 10194304:  # 4MB
        msg.reply_text('Sorry, image filesize too big')
        return
    file_ext = '.' + msg.document.mime_type.split('/', 1)[1]
    with tempfile.NamedTemporaryFile(suffix=file_ext) as f:
        file_t = bot.getFile(msg.document.file_id)
        file_t.download(custom_path=f.name)
        im = Image.open(f.name)
        logger.info("Downloaded file from chat {} as {}".format(msg.chat_id, f.name))
        size_x = str(im.size[0])
        size_y = str(im.size[1])
        logger.debug("Got image size = {},{}".format(size_x, size_y))
        reply = "Original resolution: {}x{}".format(size_x, size_y)

        try:
            # EXIF data!
            with open(f.name, 'rb') as ff:
                tags = exifread.process_file(ff)
                logger.debug("Got EXIF data = {}".format(tags))
                if tags:
                    manufacturer = tags.get("Image Make", "???")
                    model = tags.get("Image Model", "???")
                    iso = tags.get("EXIF ISOSpeedRatings", "???")

                    aperture_raw = tags.get("EXIF FNumber", "")
                    if aperture_raw:
                        aperture = float(Fraction(str(aperture_raw)))
                    else:
                        aperture = "???"

                    exposure_raw = tags.get("EXIF ExposureTime", "")
                    if exposure_raw:
                        frac = Fraction(str(exposure_raw))
                        if frac.denominator > 1000:
                            exposure = str(round(frac, 3))
                        else:
                            exposure = str(frac)
                    else:
                        exposure = "???"

                    focal_len_raw = tags.get("EXIF FocalLength", "")
                    if focal_len_raw:
                        focal_len = round(float(Fraction(str(focal_len_raw))))
                    else:
                        focal_len = "???"

                    reply += "; Model: {} {}; f: {}mm; ISO {}, f/{}, {}s".format(manufacturer, model, focal_len, iso,
                                                                                 aperture, exposure)
                else:
                    reply += "; no EXIF data :("
        except:
            reply += "; error processing EXIF data :("

        picmsg = msg.reply_photo(f, reply)
        logger.debug("Sent reply: {}".format(picmsg))

    logger.info('Replied to chat {} with "{}" and document: {}'.format(msg.chat_id, reply, msg.document))


def error(bot, update, err):
    logger.warning('Update "%s" caused error "%s"' % (update, err))


def main():
    # Create the EventHandler and pass it the bot token
    updater = Updater(os.environ.get('TG_BOT_KEY'))

    dp = updater.dispatcher

    dp.add_handler(MessageHandler(Filters.document, document_msg_handler))

    # Log all errors
    dp.add_error_handler(error)

    # Start the bot
    updater.start_polling()

    # Idle this thread
    updater.idle()


if __name__ == "__main__":
    main()
