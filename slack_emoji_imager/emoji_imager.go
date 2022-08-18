package slack_emoji_imager

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
	"log"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/draw"

	"github.com/golang/freetype/truetype"
	"github.com/pkg/errors"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/math/fixed"
)

var ErrorNoReactions = errors.Errorf("NoReactions")

const SlackAPIEndpoint = "https://slack.com/api"

const reactionEmojiSize = 50
const reactionNumSize = 50
const reactionMerginSize = 5
const laneReactionNum = 8
const emojiFilePath = "NotoColorEmoji"

var reactionWidth = reactionEmojiSize + reactionNumSize + reactionMerginSize*2
var imageWidth = reactionWidth * laneReactionNum
var imageLaneHeight = reactionMerginSize*2 + reactionEmojiSize

var colorPalette = append(palette.WebSafe, image.Transparent)

type Imager struct {
	EmojiList EmojiList
	userToken string
	botToken  string
}

type EmojiList map[string]string

type MessageReaction struct {
	Emoji string
	Num   int
}

type slackReaction struct {
	Count int    `json:"count"`
	Name  string `json:"name"`
	image struct {
		isGif     bool
		converted []*image.Paletted
		bounds    []image.Rectangle
		other     *image.Paletted
	}
}
type reactionsGetResponse struct {
	Message struct {
		Reactions []slackReaction `json:"reactions"`
	} `json:"message"`
	Error string `json:"error"`
	OK    bool   `json:"ok"`
}

func New(userToken, botToken string) (*Imager, error) {
	var imager = &Imager{
		EmojiList: make(EmojiList),
		userToken: userToken,
		botToken:  botToken,
	}
	err := imager.getEmojiList()

	return imager, err
}

func (s *Imager) MakeReactionsImage(channel string, timestamp string) (r io.Reader, err error) {
	// Get Slack Message Reactions
	reactions, err := s.getSlackReactions(channel, timestamp)
	if err != nil {
		return nil, errors.Wrap(err, "getSlackReactinos")
	}

	if len(reactions) == 0 {
		return nil, ErrorNoReactions
	}

	var maxFrame int = 1
	var nframe int

	// Get Reaction Images
	for i := range reactions {
		reactions[i], maxFrame, err = s.resize(reactions[i])
		if err != nil {
			log.Println(errors.Wrap(err, "Resize").Error())
		}

		if nframe > maxFrame {
			maxFrame = nframe
		}
	}

	// Make Reaction Image
	var gifImage *gif.GIF
	frames := make([]*image.Paletted, maxFrame)
	gifImage = &gif.GIF{
		Image: frames,
	}

	ft, err := truetype.Parse(gobold.TTF)
	if err != nil {
		return nil, errors.Wrap(err, "FontParseError")
	}

	var setEmojiToImage = func(fromFrame, toFrame int) {
		for frameNum := fromFrame; frameNum < toFrame; frameNum++ {
			var frame = image.NewPaletted(image.Rect(0, 0, imageWidth, imageLaneHeight*((len(reactions)-1)/laneReactionNum+1)), colorPalette)
			frame = s.fillFrame(frame, color.White)

			for j, reaction := range reactions {
				// draw reaction image
				var img *image.Paletted
				var bound image.Rectangle
				if reaction.image.isGif {
					// if image is gif, select a draw frame
					if frameNum >= len(reaction.image.converted) {
						// frameNum%len(reaction.image.converted) make GIF loop
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
					img.Bounds().Min.X + reactionWidth*(j%laneReactionNum) + reactionMerginSize + reactionEmojiSize/2 - img.Bounds().Dx()/2,
					img.Bounds().Min.Y + imageLaneHeight*(j/laneReactionNum) + reactionMerginSize + reactionEmojiSize/2 - img.Bounds().Dy()/2,
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
							Size: reactionNumSize,
						},
					),
					Dot: fixed.Point26_6{},
				}

				dr.Dot.X = fixed.I(reactionWidth*(j%laneReactionNum)+reactionMerginSize+reactionEmojiSize) +
					(fixed.I(reactionNumSize)-dr.MeasureString(number))/2
				dr.Dot.Y = fixed.I(reactionEmojiSize) + fixed.I(imageLaneHeight*(j/laneReactionNum))

				dr.DrawString(number)
			}

			gifImage.Image[frameNum] = frame
		}
	}

	s.paralleExec(maxFrame, setEmojiToImage)

	gifImage.Delay = make([]int, maxFrame)
	gifImage.Disposal = make([]byte, maxFrame)

	// TODO: Dynamically adjust the delay time value

	for i := range gifImage.Delay {
		gifImage.Delay[i] = 4
		gifImage.Disposal[i] = gif.DisposalBackground
	}

	var encodedGIF = new(bytes.Buffer)
	gif.EncodeAll(encodedGIF, gifImage)

	return encodedGIF, nil
}

func (s *Imager) fillFrame(frame *image.Paletted, c color.Color) *image.Paletted {
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

func (s *Imager) paralleExec(max int, execFunc func(from, to int)) {
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

func (s *Imager) getEmojiList() error {
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

func (s *Imager) AddEmoji(name string, uri string) {
	s.EmojiList[name] = uri
}

func (s *Imager) RemoveEmoji(name string) {
	delete(s.EmojiList, name)
}

func (s *Imager) GetEmojiURI(name string) string {
	var uri = s.EmojiList[name]
	if strings.HasPrefix(uri, "alias:") {
		uri = s.GetEmojiURI(strings.TrimPrefix(uri, "aslias:"))
	}

	return uri
}
