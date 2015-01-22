package minhash

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMinHasher(t *testing.T) {
	mh := New(10, 2, 2)

	mh.Add("1", strings.NewReader(`Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed sed felis vestibulum, mollis libero eget, pharetra lorem. Sed ut vestibulum tortor. Suspendisse sem nisl, semper eu sem non, tempor viverra ante. Morbi quis nunc non orci fermentum fringilla sit amet nec nisi. Morbi laoreet commodo porta. Ut bibendum porttitor bibendum. Nulla scelerisque eu sem at efficitur. Quisque a imperdiet massa.`))
	mh.Add("2", strings.NewReader(`Nulla dapibus lorem nunc, nec tempus purus dictum vel. Nullam lacinia ultricies cursus. Ut quis lectus efficitur, porta dolor nec, ornare tellus. Nunc felis orci, scelerisque mollis elementum sed, laoreet mollis sem. Sed sollicitudin massa ultricies ultricies hendrerit. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nullam finibus lobortis commodo. In dignissim urna a neque lacinia mattis.`))

	dissimlarText := strings.NewReader(`Cras gravida bibendum venenatis. Nulla tempus ante eget rutrum maximus. Pellentesque vel lorem nisi. Nullam varius neque sed lectus feugiat, ac vestibulum nisi porttitor. Sed risus nisi, ultrices in nisi vitae, convallis congue dolor. Aenean tempor justo quis nisi maximus malesuada. Duis fermentum justo sem, a feugiat velit sagittis eget.`)
	results := mh.FindSimilar(dissimlarText, 0)

	assert.Len(t, results, 0)

	similarText := strings.NewReader(`Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed sed felis vestibulum, mollis libero eget, pharetra lorem. Sed ut vestibulum tortor. Suspendisse sem nisl, semper eu sem non, tempor viverra ante. Morbi quis nunc non orci fermentum fringilla sit amet nec nisi. Morbi laoreet commodo porta. Ut bibendum porttitor bibendum. Blah nulla scelerisque eu sem at efficitur. Quisque a imperdiet massa.`)
	results = mh.FindSimilar(similarText, .8)

	assert.Len(t, results, 1)
	assert.Equal(t, results[0].ID, "1")

	assert.True(t, mh.Contains("1"))
	assert.True(t, mh.Contains("2"))
	assert.False(t, mh.Contains("3"))
}
