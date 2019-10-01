package kinesis

import (
	"os"
	"testing"

	"github.com/Flaque/filet"
	"github.com/influxdata/telegraf/testutil"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestPartitionKey(t *testing.T) {

	assert := assert.New(t)
	testPoint := testutil.TestMetric(1)

	k := KinesisOutput{
		Partition: &Partition{
			Method: "static",
			Key:    "-",
		},
	}
	assert.Equal("-", k.getPartitionKey(testPoint), "PartitionKey should be '-'")

	k = KinesisOutput{
		Partition: &Partition{
			Method: "tag",
			Key:    "tag1",
		},
	}
	assert.Equal(testPoint.Tags()["tag1"], k.getPartitionKey(testPoint), "PartitionKey should be value of 'tag1'")

	k = KinesisOutput{
		Partition: &Partition{
			Method:  "tag",
			Key:     "doesnotexist",
			Default: "somedefault",
		},
	}
	assert.Equal("somedefault", k.getPartitionKey(testPoint), "PartitionKey should use default")

	k = KinesisOutput{
		Partition: &Partition{
			Method: "tag",
			Key:    "doesnotexist",
		},
	}
	assert.Equal("telegraf", k.getPartitionKey(testPoint), "PartitionKey should be telegraf")

	k = KinesisOutput{
		Partition: &Partition{
			Method: "not supported",
		},
	}
	assert.Equal("", k.getPartitionKey(testPoint), "PartitionKey should be value of ''")

	k = KinesisOutput{
		Partition: &Partition{
			Method: "measurement",
		},
	}
	assert.Equal(testPoint.Name(), k.getPartitionKey(testPoint), "PartitionKey should be value of measurement name")

	k = KinesisOutput{
		Partition: &Partition{
			Method: "random",
		},
	}
	partitionKey := k.getPartitionKey(testPoint)
	u, err := uuid.FromString(partitionKey)
	assert.Nil(err, "Issue parsing UUID")
	assert.Equal(byte(4), u.Version(), "PartitionKey should be UUIDv4")
}

func TestConfigDecoding(t *testing.T) {
	ko, err := NewKinesisOutput("./kinesis_output_test.toml")
	if err != nil {
		t.Logf("Failed to make new Kinesis Output. Error: %s", err)
		t.Fail()
	}
	t.Logf("Shard count: %d", ko.OverrideShardCount)
}

func TestConfigEnvVars(t *testing.T) {
	defer filet.CleanUp(t)
	expectStreamName := "TestStreamValue"

	tmpContents := `
region = "ap-southeast-2"
streamname = "${TESTING_KINESIS_STREAM_NAME}"
`
	testingConfig := filet.TmpFile(t, "", tmpContents)

	err := os.Setenv("TESTING_KINESIS_STREAM_NAME", expectStreamName)
	if err != nil {
		t.Logf("Couldn't set the testing Environment variable. Error: %s", err)
		t.Fail()
	}

	ko, err := NewKinesisOutput(testingConfig.Name())

	if err != nil {
		t.Logf("Failed to make new Kinesis Output. Error: %s", err)
		t.Fail()
	}

	if ko.StreamName != expectStreamName {
		t.Logf("StreamName from Environment variable was not found. Want: %s, Got: %s", expectStreamName, ko.StreamName)
		t.Fail()
	}
}
