#!/bin/sh

convert -size 256x256 -resize 256x256 -extent 256x256 -gravity center -fuzz 30% -transparent white static/Logo.svg static/favicon.ico
convert -size 256x256 -resize 256x256 -extent 256x256 -gravity center -fuzz 30% -transparent white static/Star.svg static/faviconStar.ico
