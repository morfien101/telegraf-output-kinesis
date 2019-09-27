package kinesis

import (
	"io/ioutil"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/serializers"
	"github.com/influxdata/toml"
	encoders "github.com/morfien101/telegraf-encoders"
	"github.com/morfien101/telegraf-output-kinesis/awsCreds"
	"github.com/morfien101/telegraf-output-kinesis/serializerclone"
	uuid "github.com/satori/go.uuid"
)

// KinesisOutput describes a telegraf kinesis output plugin
type KinesisOutput struct {
	Region      string `toml:"region"`
	AccessKey   string `toml:"access_key"`
	SecretKey   string `toml:"secret_key"`
	RoleARN     string `toml:"role_arn"`
	Profile     string `toml:"profile"`
	Filename    string `toml:"shared_credential_file"`
	Token       string `toml:"token"`
	EndpointURL string `toml:"endpoint_url"`

	StreamName         string                  `toml:"streamname"`
	Partition          *Partition              `toml:"partition"`
	Debug              bool                    `toml:"debug"`
	AggregateMetrics   bool                    `toml:"aggregate_metrics"`
	OverrideShardCount int                     `toml:"override_shard_count"`
	ContentEncoding    string                  `toml:"content_encoding"`
	Formatting         *serializerclone.Config `toml:"formatting"`

	svc     *kinesis.Kinesis
	nShards int64

	serializer serializers.Serializer

	encoder encoders.ContentEncoder
}

// Partition is used to detect what type of partition key you would like to use.
type Partition struct {
	Method  string `toml:"method"`
	Key     string `toml:"key"`
	Default string `toml:"default"`
}

var sampleConfig = `
  ## Amazon REGION of kinesis endpoint.
  region = "ap-southeast-2"

  ## Amazon Credentials
  ## Credentials are loaded in the following order
  ## 1) Assumed credentials via STS if role_arn is specified	
  ## 2) explicit credentials from 'access_key' and 'secret_key'
  ## 3) shared profile from 'profile'
  ## 4) environment variables
  ## 5) shared credentials file
  ## 6) EC2 Instance Profile
  #access_key = ""
  #secret_key = ""
  #token = ""
  #role_arn = ""
  #profile = ""
  #shared_credential_file = ""

  ## Endpoint to make request against, the correct endpoint is automatically
  ## determined and this option should only be set if you wish to override the
  ## default.
  ##   ex: endpoint_url = "http://localhost:8000"
  # endpoint_url = ""

  ## Kinesis StreamName must exist prior to starting telegraf.
  streamname = "StreamName"
  ## DEPRECATED: PartitionKey as used for sharding data.
  partitionkey = "PartitionKey"
  ## DEPRECATED: If set the paritionKey will be a random UUID on every put.
  ## This allows for scaling across multiple shards in a stream.
  ## This will cause issues with ordering.
  use_random_partitionkey = false
  ## The partition key can be calculated using one of several methods:
  ##
  ## Use a static value for all writes:
  #  [outputs.kinesis.partition]
  #    method = "static"
  #    key = "howdy"
  #
  ## Use a random partition key on each write:
  #  [outputs.kinesis.partition]
  #    method = "random"
  #
  ## Use the measurement name as the partition key:
  #  [outputs.kinesis.partition]
  #    method = "measurement"
  #
  ## Use the value of a tag for all writes, if the tag is not set the empty
  ## default option will be used. When no default, defaults to "telegraf"
  #  [outputs.kinesis.partition]
  #    method = "tag"
  #    key = "host"
  #    default = "mykey"


  ## Data format to output.
  ## Each data format has its own unique set of configuration options, read
  ## more about them here:
  ## https://github.com/influxdata/telegraf/blob/master/docs/DATA_FORMATS_OUTPUT.md
	data_format = "influx"
	
	# Aggregate metrics into payloads that will fit into the Kinesis records. This
	# is designed to save money by making more efficient use of Kinesis.
	aggregate_metrics = true

	# Kinesis cares little for what you send into it via the records.
	# We can therefore save more money by compressing the aggregated metrics.
	# Note, this only works with the aggregated metrics set to true.
	# valid options: "gzip", "snappy"
	# See https://github.com/influxdata/telegraf/tree/master/plugins/outputs/kinesis
	# for more details on each compression method.
	content_encoding = "gzip"

  ## debug will show upstream aws messages.
  debug = false
`

func (k *KinesisOutput) SampleConfig() string {
	return sampleConfig
}

func (k *KinesisOutput) Description() string {
	return "Configuration for the AWS Kinesis output."
}

func NewKinesisOutput(configFilePath string) (*KinesisOutput, error) {
	cfgBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}

	ko := &KinesisOutput{}
	err = toml.Unmarshal(cfgBytes, ko)
	if err != nil {
		return nil, err
	}

	serializerConfig := serializerclone.NewTelegrafConfig(ko.Formatting)
	serializer, err := serializers.NewSerializer(serializerConfig)
	if err != nil {
		return nil, err
	}

	ko.SetSerializer(serializer)
	return ko, nil
}

// Connect will establish a connection to AWS Kinesis and make sure there are shards ready to collect data.
func (k *KinesisOutput) Connect() error {
	// We attempt first to create a session to Kinesis using an IAMS role, if that fails it will fall through to using
	// environment variables, and then Shared Credentials.
	if k.Debug {
		log.Printf("I! kinesis: Establishing a connection to Kinesis in %s", k.Region)
	}

	encoder, err := makeEncoder(k.ContentEncoding)
	if err != nil {
		return err
	}
	k.encoder = encoder

	credentialConfig := &awsCreds.CredentialConfig{
		Region:      k.Region,
		AccessKey:   k.AccessKey,
		SecretKey:   k.SecretKey,
		RoleARN:     k.RoleARN,
		Profile:     k.Profile,
		Filename:    k.Filename,
		Token:       k.Token,
		EndpointURL: k.EndpointURL,
	}
	configProvider := credentialConfig.Credentials()
	svc := kinesis.New(configProvider)

	describeOutput, err := svc.DescribeStreamSummary(&kinesis.DescribeStreamSummaryInput{
		StreamName: aws.String(k.StreamName),
	})
	if err != nil {
		return err
	}
	k.svc = svc
	k.nShards = *describeOutput.StreamDescriptionSummary.OpenShardCount
	return nil
}

func (k *KinesisOutput) Close() error {
	return nil
}

func (k *KinesisOutput) SetSerializer(serializer serializers.Serializer) {
	k.serializer = serializer
}

func writekinesis(k *KinesisOutput, r []*kinesis.PutRecordsRequestEntry) time.Duration {
	start := time.Now()
	payload := &kinesis.PutRecordsInput{
		Records:    r,
		StreamName: aws.String(k.StreamName),
	}

	resp, err := k.svc.PutRecords(payload)
	if err != nil {
		log.Printf("E! kinesis: Unable to write to Kinesis : %s", err.Error())
	}
	if k.Debug {
		log.Printf("I! Wrote: '%+v'", resp)
	}
	return time.Since(start)
}

func (k *KinesisOutput) getPartitionKey(metric telegraf.Metric) string {
	randomKey := func() string {
		if k.AggregateMetrics {
			return randomPartitionKey
		}

		u := uuid.NewV4()
		return u.String()
	}

	switch k.Partition.Method {
	case "static":
		return k.Partition.Key
	case "random":
		return randomKey()
	case "measurement":
		return metric.Name()
	case "tag":
		if t, ok := metric.GetTag(k.Partition.Key); ok {
			return t
		} else if len(k.Partition.Default) > 0 {
			return k.Partition.Default
		}
		// Default partition name if default is not set
		return "telegraf"
	default:
		log.Printf("E! kinesis : You have configured a Partition method of '%s' which is not supported", k.Partition.Method)
	}

	return "telegraf"
}

func (k *KinesisOutput) Write(metrics []telegraf.Metric) error {
	if len(metrics) == 0 {
		return nil
	}

	switch {
	case k.AggregateMetrics:
		return k.aggregatedWrite(metrics)
	default:
		return k.writeDefault(metrics)
	}
}

func (k *KinesisOutput) writeDefault(metrics []telegraf.Metric) error {
	var sz uint32

	r := []*kinesis.PutRecordsRequestEntry{}

	for _, metric := range metrics {
		sz++

		values, err := k.serializer.Serialize(metric)
		if err != nil {
			log.Printf("D! [outputs.kinesis] Could not serialize metric: %v\n", err)
			continue
		}

		partitionKey := k.getPartitionKey(metric)

		d := kinesis.PutRecordsRequestEntry{
			Data:         values,
			PartitionKey: aws.String(partitionKey),
		}

		r = append(r, &d)

		if sz == 500 {
			// Max Messages Per PutRecordRequest is 500
			elapsed := writekinesis(k, r)
			log.Printf("D! Wrote a %d point batch to Kinesis in %+v.\n", sz, elapsed)
			sz = 0
			r = nil
		}

	}
	if sz > 0 {
		elapsed := writekinesis(k, r)
		log.Printf("D! Wrote a %d point batch to Kinesis in %+v.\n", sz, elapsed)
	}

	return nil
}

func (k *KinesisOutput) aggregatedWrite(metrics []telegraf.Metric) error {
	log.Printf("D! Starting aggregated writer with %d metrics.\n", len(metrics))

	handler := newPutRecordsHandler(k.serializer)

	for _, metric := range metrics {
		err := handler.addRawMetric(k.getPartitionKey(metric), metric)
		if err != nil {
			return err
		}
	}
	handler.packageMetrics(k.nShards)

	// encode the messages if required.
	err := handler.encodePayloadBodies(k.encoder)
	if err != nil {
		return err
	}

	var elapsed time.Duration
	for _, writeRequests := range handler.convertToKinesisPutRequests() {
		t := writekinesis(k, writeRequests)
		elapsed = elapsed + t
	}

	log.Printf("D! Wrote aggregated metrics in %+v.\n", elapsed)

	return nil
}

func newGzipEncoder() (*encoders.GzipEncoder, error) {
	// Grab the Gzip encoder directly because we need to set the level.
	gz, err := encoders.NewGzipEncoder()
	if err != nil {
		return nil, err
	}
	err = gz.SetLevel(gzipCompressionLevel)
	if err != nil {
		return nil, err
	}

	return gz, nil
}

func makeEncoder(encoderType string) (encoders.ContentEncoder, error) {
	switch encoderType {
	case "gzip":
		// Special handling for gzip because we need to change the level of compression.
		return newGzipEncoder()
	default:
		return encoders.NewContentEncoder(encoderType)
	}
}
