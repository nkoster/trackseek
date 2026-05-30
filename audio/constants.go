package audio

const (
	// defaultTargetSampleRate is the fallback output rate used when no env override is set.
	defaultTargetSampleRate = 44100

	// pcm16FullScale normalizes signed 16-bit PCM samples into the float range used internally.
	pcm16FullScale = 32768.0

	// mp3DecoderBytesPerSample matches the 16-bit PCM samples returned by the MP3 decoder.
	mp3DecoderBytesPerSample = 2

	// mp3DecoderChannels assumes the decoder output is interleaved stereo PCM.
	mp3DecoderChannels = 2

	// lowPassFilterTaps controls the FIR filter length used before downsampling.
	lowPassFilterTaps = 63

	// lowPassCutoffRatio leaves some guard band below Nyquist to reduce aliasing.
	lowPassCutoffRatio = 0.45

	// hammingWindowAlpha and hammingWindowBeta define the Hamming window coefficients.
	hammingWindowAlpha = 0.54
	hammingWindowBeta  = 0.46
)
