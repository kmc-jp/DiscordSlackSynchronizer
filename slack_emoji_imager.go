package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/draw"

	"github.com/golang/freetype/truetype"
	"github.com/kyokomi/emoji"
	"github.com/nfnt/resize"
	"github.com/pkg/errors"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/math/fixed"
)

var SlackEmojiImagerErrorNoReactions = errors.Errorf("NoReactions")

type SlackEmojiImager struct {
	EmojiList SlackEmojiList
	userToken string
	botToken  string
}

type SlackEmojiList map[string]string

type SlackMessageReaction struct {
	Emoji string
	Num   int
}

func NewSlackEmojiImager(userToken, botToken string) (*SlackEmojiImager, error) {
	var imager = &SlackEmojiImager{
		EmojiList: make(SlackEmojiList),
		userToken: userToken,
		botToken:  botToken,
	}
	err := imager.getEmojiList()

	return imager, err
}

func (s *SlackEmojiImager) MakeReactionsImage(channel string, timestamp string) (r io.Reader, err error) {
	const ReactionEmojiSize = 30
	const ReactionNumSize = 30
	const ReactionMerginSize = 5
	const LaneReactionNum = 5
	const EmojiFilePath = "NotoColorEmoji"

	var colorPalette = append(palette.WebSafe, image.Transparent)

	var ReactionWidth = ReactionEmojiSize + ReactionNumSize + ReactionMerginSize*2
	var ImageWidth = ReactionWidth * 5
	var ImageLaneHeight = ReactionMerginSize*2 + ReactionEmojiSize

	// Get Slack Message Reactions
	type reactionsGetResponse struct {
		Message struct {
			Reactions []struct {
				Count int    `json:"count"`
				Name  string `json:"name"`
				image struct {
					isGif     bool
					converted []*image.Paletted
					bounds    []image.Rectangle
					other     *image.Paletted
				}
			} `json:"reactions"`
		} `json:"message"`
		Error string `json:"error"`
		OK    bool   `json:"ok"`
	}
	var requestAttr = make(url.Values)

	requestAttr.Add("channel", channel)
	requestAttr.Add("timestamp", timestamp)

	var client = http.DefaultClient
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/reactions.get?%s", SlackAPIEndpoint, requestAttr.Encode()), nil)
	if err != nil {
		return
	}

	req.Header.Set("Content-type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "Bearer "+s.botToken)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var responseAttr reactionsGetResponse
	err = json.NewDecoder(resp.Body).Decode(&responseAttr)
	if err != nil {
		return
	}

	if !responseAttr.OK {
		return nil, fmt.Errorf("GetMessageReactionsError: %s", responseAttr.Error)
	}

	var reactions = responseAttr.Message.Reactions

	if len(reactions) == 0 {
		return nil, SlackEmojiImagerErrorNoReactions
	}

	var maxFrame int = 1

	// Get Reaction Images
	for i := range reactions {
		var uri = s.GetEmojiURI(reactions[i].Name)

		switch {
		case uri != "":
			// custom emoji
			var client = http.DefaultClient
			req, err = http.NewRequest("GET", uri, nil)
			if err != nil {
				fmt.Println(err)
				continue
			}

			req.Header.Set("Authorization", "Bearer "+s.botToken)

			resp, err = client.Do(req)
			if err != nil {
				fmt.Println(err)
				continue
			}
			defer resp.Body.Close()

			switch {
			case strings.HasSuffix(uri, ".gif"):
				// isGif
				gifImage, err := gif.DecodeAll(resp.Body)
				if err != nil {
					fmt.Println(err)
					continue
				}

				if len(gifImage.Image) > maxFrame {
					maxFrame = len(gifImage.Image)
				}

				reactions[i].image.converted = make([]*image.Paletted, len(gifImage.Image))
				reactions[i].image.bounds = make([]image.Rectangle, len(gifImage.Image))

				reactions[i].image.isGif = true

				var width, height = float64(gifImage.Config.Width), float64(gifImage.Config.Height)

				var ratio float64
				if width > height {
					ratio = float64(ReactionEmojiSize) / width
				} else {
					ratio = float64(ReactionEmojiSize) / height
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

							reactions[i].image.bounds[frameNum] = image.Rect(
								marginX, marginY,
								resizedBounds.Dx()+marginX,
								resizedBounds.Dy()+marginY,
							)

							resizedPaletted := image.NewPaletted(reactions[i].image.bounds[frameNum], colorPalette)
							if haveBackgound {
								resizedPaletted.ColorModel().Convert(bColor)
								resizedPaletted = s.fillFrame(resizedPaletted, bColor)
							}

							draw.FloydSteinberg.Draw(resizedPaletted, reactions[i].image.bounds[frameNum], resizedImage, image.Point{})
							reactions[i].image.converted[frameNum] = resizedPaletted
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
					continue
				}

				var width, height = float64(srcImage.Bounds().Size().X), float64(srcImage.Bounds().Size().Y)

				var ratio float64
				if width > height {
					ratio = float64(ReactionEmojiSize) / width
				} else {
					ratio = float64(ReactionEmojiSize) / height
				}
				srcImage = resize.Resize(
					uint(math.Floor(width*ratio)),
					uint(math.Floor(height*ratio)),
					srcImage, resize.Lanczos3,
				)
				resizedPaletted := image.NewPaletted(srcImage.Bounds(), colorPalette)
				draw.FloydSteinberg.Draw(resizedPaletted, srcImage.Bounds(), srcImage, image.Point{})
				reactions[i].image.other = resizedPaletted
			}

		case uri == "":
			// default emoji
			var emojiStr = emoji.Sprintf(":%s:", reactions[i].Name)
			if emojiStr == fmt.Sprintf(":%s:", reactions[i].Name) {
				// emoji not found
				continue
			}

			var emojiRune = []rune(emojiStr)

			var path = filepath.Join(EmojiFilePath, fmt.Sprintf("emoji_u%x.png", emojiRune[0]))
			fp, err := os.Open(path)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			defer fp.Close()

			srcImage, _, err := image.Decode(fp)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}

			var width, height = float64(srcImage.Bounds().Size().X), float64(srcImage.Bounds().Size().Y)

			var ratio float64
			if width > height {
				ratio = float64(ReactionEmojiSize) / width
			} else {
				ratio = float64(ReactionEmojiSize) / height
			}
			srcImage = resize.Resize(
				uint(math.Floor(width*ratio)),
				uint(math.Floor(height*ratio)),
				srcImage, resize.Lanczos3,
			)

			resizedPaletted := image.NewPaletted(srcImage.Bounds(), colorPalette)
			draw.FloydSteinberg.Draw(resizedPaletted, srcImage.Bounds(), srcImage, image.Point{})

			reactions[i].image.other = resizedPaletted
		}
	}

	// Make Reaction Image
	var gifImage *gif.GIF
	frames := make([]*image.Paletted, maxFrame)
	gifImage = &gif.GIF{
		Image: frames,
	}

	{
		ft, err := truetype.Parse(gobold.TTF)
		if err != nil {
			return nil, err
		}

		var encodeGIF = func(fromFrame, toFrame int) {
			for frameNum := fromFrame; frameNum < toFrame; frameNum++ {
				var frame = image.NewPaletted(image.Rect(0, 0, ImageWidth, ImageLaneHeight*((len(reactions)-1)/LaneReactionNum+1)), colorPalette)
				frame = s.fillFrame(frame, color.White)

				for j, reaction := range reactions {
					// draw reaction image
					var img *image.Paletted
					var bound image.Rectangle
					if reaction.image.isGif {
						// if image is gif, select a draw frame
						if frameNum >= len(reaction.image.converted) {
							img = reaction.image.converted[frameNum%len(reaction.image.converted)]
							bound = reaction.image.bounds[frameNum%len(reaction.image.converted)]
						} else {
							img = reaction.image.converted[frameNum]
							bound = reaction.image.bounds[frameNum]
						}
					} else {
						img = reaction.image.other
						if img == nil {
							continue
						}
						bound = img.Bounds()
					}

					var imgPoint = image.Point{
						img.Bounds().Min.X + ReactionWidth*(j%LaneReactionNum) + ReactionMerginSize,
						img.Bounds().Min.Y + ImageLaneHeight*(j/LaneReactionNum) + ReactionMerginSize,
					}

					draw.Copy(frame, imgPoint, img, bound, draw.Over, nil)

					// draw reaction number
					var number = strconv.Itoa(reaction.Count)

					var dr = &font.Drawer{
						Dst: frame,
						Src: image.Black,
						Face: truetype.NewFace(
							ft,
							&truetype.Options{
								Size: ReactionNumSize,
							},
						),
						Dot: fixed.Point26_6{},
					}

					dr.Dot.X = fixed.I(ReactionWidth*(j%LaneReactionNum)+ReactionMerginSize+ReactionEmojiSize) +
						(fixed.I(ReactionNumSize)-dr.MeasureString(number))/2
					dr.Dot.Y = fixed.I(ReactionEmojiSize) + fixed.I(ImageLaneHeight*(j/LaneReactionNum))

					dr.DrawString(number)
				}

				gifImage.Image[frameNum] = frame
			}
		}

		s.paralleExec(maxFrame, encodeGIF)

	}

	gifImage.Delay = make([]int, maxFrame)
	gifImage.Disposal = make([]byte, maxFrame)

	for i := range gifImage.Delay {
		gifImage.Delay[i] = 4
		gifImage.Disposal[i] = gif.DisposalBackground
	}

	var encodedGIF = new(bytes.Buffer)
	gif.EncodeAll(encodedGIF, gifImage)

	return encodedGIF, nil
}

func (s *SlackEmojiImager) fillFrame(frame *image.Paletted, c color.Color) *image.Paletted {
	var rect = frame.Rect
	var newFrame = &image.Paletted{}

	*newFrame = *frame

	for h := rect.Min.Y; h < rect.Max.Y; h++ {
		for v := rect.Min.X; v < rect.Max.X; v++ {
			newFrame.Set(v, h, c)
		}
	}

	return newFrame
}
func (s *SlackEmojiImager) paralleExec(max int, execFunc func(from, to int)) {
	var cpus = runtime.NumCPU()
	var wg sync.WaitGroup

	var rest = max % cpus
	var from, to int = 0, 0

	for cpu := 0; cpu < cpus; cpu++ {
		if rest > 0 {
			to = from + max/cpus + 1
			rest--
		} else {
			to = from + max/cpus
		}

		if to == from {
			break
		}

		wg.Add(1)

		go func(from, to int) {
			defer wg.Done()
			execFunc(from, to)
		}(from, to)

		from = to
	}

	wg.Wait()
}

func (s *SlackEmojiImager) getEmojiList() error {
	type emojiListAPIResponse struct {
		OK    bool              `json:"ok"`
		Error string            `json:"error"`
		Emoji map[string]string `json:"emoji"`
	}

	var err error
	var requestAttr = make(url.Values)

	req, err := http.NewRequest("GET", SlackAPIEndpoint+"/emoji.list", strings.NewReader(requestAttr.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "Bearer "+s.userToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var responseAttr emojiListAPIResponse
	err = json.NewDecoder(resp.Body).Decode(&responseAttr)
	if err != nil {
		return err
	}

	if !responseAttr.OK {
		return fmt.Errorf("EmojiListGetError")
	}

	s.EmojiList = responseAttr.Emoji

	return err
}

func (s *SlackEmojiImager) AddEmoji(name string, uri string) {
	s.EmojiList[name] = uri
}

func (s *SlackEmojiImager) RemoveEmoji(name string) {
	delete(s.EmojiList, name)
}

func (s *SlackEmojiImager) GetEmojiURI(name string) string {
	var uri = s.EmojiList[name]
	if strings.HasPrefix(uri, "alias:") {
		uri = s.GetEmojiURI(strings.TrimPrefix(uri, "aslias:"))
	}

	return uri
}
