package serializerclone

import (
	"time"

	"github.com/influxdata/telegraf/plugins/serializers"
)

// Config is used to pass in the values to the telegraf serializer config
// while also giving us the ability to use toml to specify the values.
type Config struct {
	// Dataformat can be one of the serializer types listed in NewSerializer.
	DataFormat string `tom:"data_format"`

	// Support tags in graphite protocol
	GraphiteTagSupport bool `tom:"graphite_tag_support"`

	// Maximum line length in bytes; influx format only
	InfluxMaxLineBytes int `tom:"influx_max_line_bytes"`

	// Sort field keys, set to true only when debugging as it less performant
	// than unsorted fields; influx format only
	InfluxSortFields bool `tom:"influx_sort_fields"`

	// Support unsigned integer output; influx format only
	InfluxUintSupport bool `tom:"influx_uint_support"`

	// Prefix to add to all measurements, only supports Graphite
	Prefix string `tom:"prefix"`

	// Template for converting telegraf metrics into Graphite
	// only supports Graphite
	Template string `tom:"template"`

	// Timestamp units to use for JSON formatted output
	TimestampUnits time.Duration `tom:"json_timestamp_units"`

	// Include HEC routing fields for splunkmetric output
	HecRouting bool `tom:"splunkmetric_hec_routing"`

	// Point tags to use as the source name for Wavefront (if none found, host will be used).
	WavefrontSourceOverride []string `tom:"wavefront_source_override"`

	// Use Strict rules to sanitize metric and tag names from invalid characters for Wavefront
	// When enabled forward slash (/) and comma (,) will be accepted
	WavefrontUseStrict bool `tom:"wavefront_use_strict"`
}

// New will return a clone Config that has the default data format of influx
func New() *Config {
	return &Config{DataFormat: "influx"}
}

// NewTelegrafConfig will return a serializer config that uses the config passed in to create it.
func NewTelegrafConfig(clone *Config) *serializers.Config {
	tc := &serializers.Config{
		DataFormat:              clone.DataFormat,
		GraphiteTagSupport:      clone.GraphiteTagSupport,
		InfluxMaxLineBytes:      clone.InfluxMaxLineBytes,
		InfluxSortFields:        clone.InfluxSortFields,
		InfluxUintSupport:       clone.InfluxUintSupport,
		Prefix:                  clone.Prefix,
		Template:                clone.Template,
		TimestampUnits:          clone.TimestampUnits,
		HecRouting:              clone.HecRouting,
		WavefrontSourceOverride: clone.WavefrontSourceOverride,
		WavefrontUseStrict:      clone.WavefrontUseStrict,
	}
	if tc.DataFormat == "" {
		tc.DataFormat = "influx"
	}
	return tc
}
