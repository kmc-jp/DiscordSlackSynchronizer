package slack_emoji_imager

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kyokomi/emoji"
	"github.com/nfnt/resize"
	"github.com/pkg/errors"
	"golang.org/x/image/draw"
)

func (s *Imager) resize(reaction slackReaction) (resizedReaction slackReaction, maxFrame int, err error) {
	var uri = s.GetEmojiURI(reaction.Name)

	switch {
	case uri != "":
		// custom emoji
		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			return reaction, 0, errors.Wrap(err, "makeRequestCustomEmojiImage")
		}

		req.Header.Set("Authorization", "Bearer "+s.botToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return reaction, 0, errors.Wrap(err, "requestCustomEmojiImage")
		}
		defer resp.Body.Close()

		switch {
		case strings.HasSuffix(uri, ".gif"):
			// isGif
			gifImage, err := gif.DecodeAll(resp.Body)
			if err != nil {
				return reaction, 0, errors.Wrap(err, "DecodeGif")
			}

			if len(gifImage.Image) > maxFrame {
				maxFrame = len(gifImage.Image)
			}

			reaction.image.converted = make([]*image.Paletted, len(gifImage.Image))
			reaction.image.bounds = make([]image.Rectangle, len(gifImage.Image))

			reaction.image.isGif = true

			var width, height = float64(gifImage.Config.Width), float64(gifImage.Config.Height)

			var ratio float64
			if width > height {
				ratio = float64(reactionEmojiSize) / width
			} else {
				ratio = float64(reactionEmojiSize) / height
			}

			{
				// First, get backgound color
				var bColor color.Color
				var haveBackgound bool

				p, haveBackgound := gifImage.Config.ColorModel.(color.Palette)
				if haveBackgound {
					bColor = p[int(gifImage.BackgroundIndex)]
					bColor = color.Palette(colorPalette).Convert(bColor)
				}

				// Then, resize GIF frames
				var resizeGIF = func(fromFrame, toFrame int) {
					for frameNum := fromFrame; frameNum < toFrame; frameNum++ {
						var frame = gifImage.Image[frameNum]
						var rect = frame.Bounds()
						var tmpImage = frame.SubImage(rect)
						var resizedImage = resize.Resize(
							uint(math.Floor(float64(rect.Dx())*ratio)),
							uint(math.Floor(float64(rect.Dy())*ratio)),
							tmpImage, resize.Lanczos3,
						)

						var resizedBounds = resizedImage.Bounds()

						marginX := int(math.Floor(float64(rect.Min.X) * ratio))
						marginY := int(math.Floor(float64(rect.Min.Y) * ratio))

						reaction.image.bounds[frameNum] = image.Rect(
							marginX, marginY,
							resizedBounds.Dx()+marginX,
							resizedBounds.Dy()+marginY,
						)

						resizedPaletted := image.NewPaletted(reaction.image.bounds[frameNum], colorPalette)
						if haveBackgound {
							resizedPaletted.ColorModel().Convert(bColor)
							resizedPaletted = s.fillFrame(resizedPaletted, bColor)
						}

						draw.FloydSteinberg.Draw(resizedPaletted, reaction.image.bounds[frameNum], resizedImage, image.Point{})
						reaction.image.converted[frameNum] = resizedPaletted
					}
				}

				s.paralleExec(len(gifImage.Image), resizeGIF)

				gifImage.Delay = make([]int, maxFrame)
				for i := range gifImage.Delay {
					gifImage.Delay[i] = 4
				}
			}
		default:
			// resize png, jpg
			srcImage, _, err := image.Decode(resp.Body)
			if err != nil {
				return reaction, 0, errors.Wrap(err, "DecodeImage")
			}

			var width, height = float64(srcImage.Bounds().Size().X), float64(srcImage.Bounds().Size().Y)

			var ratio float64
			if width > height {
				ratio = float64(reactionEmojiSize) / width
			} else {
				ratio = float64(reactionEmojiSize) / height
			}
			srcImage = resize.Resize(
				uint(math.Floor(width*ratio)),
				uint(math.Floor(height*ratio)),
				srcImage, resize.Lanczos3,
			)
			resizedPaletted := image.NewPaletted(srcImage.Bounds(), colorPalette)
			draw.FloydSteinberg.Draw(resizedPaletted, srcImage.Bounds(), srcImage, image.Point{})
			reaction.image.other = resizedPaletted

			if 1 > maxFrame {
				maxFrame = 1
			}
		}

	case uri == "":
		// default emoji
		var emojiStr = emoji.Sprintf(":%s:", reaction.Name)
		if emojiStr == fmt.Sprintf(":%s:", reaction.Name) {
			// emoji not found
			return reaction, 0, errors.New("DefaultEmojiNotFound")
		}

		var emojiRune = []rune(emojiStr)

		var path = filepath.Join(emojiFilePath, fmt.Sprintf("emoji_u%x.png", emojiRune[0]))
		fp, err := os.Open(path)
		if err != nil {
			return reaction, 0, errors.New("EmojiFileOpen")
		}
		defer fp.Close()

		srcImage, _, err := image.Decode(fp)
		if err != nil {
			return reaction, 0, errors.New("DecodeImage")
		}

		var width, height = float64(srcImage.Bounds().Size().X), float64(srcImage.Bounds().Size().Y)

		var ratio float64
		if width > height {
			ratio = float64(reactionEmojiSize) / width
		} else {
			ratio = float64(reactionEmojiSize) / height
		}
		srcImage = resize.Resize(
			uint(math.Floor(width*ratio)),
			uint(math.Floor(height*ratio)),
			srcImage, resize.Lanczos3,
		)

		resizedPaletted := image.NewPaletted(srcImage.Bounds(), colorPalette)
		draw.FloydSteinberg.Draw(resizedPaletted, srcImage.Bounds(), srcImage, image.Point{})

		reaction.image.other = resizedPaletted
		if 1 > maxFrame {
			maxFrame = 1
		}
	}

	return reaction, maxFrame, nil
}
