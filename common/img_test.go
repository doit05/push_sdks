package common

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDowlodPic(t *testing.T) {
	url := "http://xs-image.oss-cn-hangzhou.aliyuncs.com/202009/01/100011834_5f4d45c6276fd2.59812999.jpg"
	Convey("download img success", t, func() {
		filePath, err := DowlodPic(url, os.TempDir(), "", GetImgNameFromUrl(url))
		t.Log(err)
		So(err, ShouldBeNil)
		So(filePath, ShouldNotBeEmpty)
		file, err := os.Open(filePath)
		defer file.Close()
		So(err, ShouldBeNil)
		So(file, ShouldNotBeNil)
		t.Log(filePath)
		err = os.Remove(filePath)
		So(err, ShouldBeNil)
	})

}
