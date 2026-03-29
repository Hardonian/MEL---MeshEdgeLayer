package platform

type NoCompression struct{}

func (NoCompression) Name() string                        { return "none" }
func (NoCompression) Encode(input []byte) ([]byte, error) { return input, nil }

type StandardQuantization struct{}

func (StandardQuantization) Name() string                        { return "standard_quantization" }
func (StandardQuantization) Encode(input []byte) ([]byte, error) { return input, nil }

type ExperimentalTurboQuantCompatible struct{}

func (ExperimentalTurboQuantCompatible) Name() string                        { return "experimental_turboquant_compatible" }
func (ExperimentalTurboQuantCompatible) Encode(input []byte) ([]byte, error) { return input, nil }
