package fingerprint

const (
	// windowSize is the FFT frame size used for spectral analysis.
	windowSize = 4096

	// hopSize is the frame advance between consecutive FFT windows.
	hopSize = 2048

	// maxPeaksPerFrame limits how many dominant peaks are kept from each frame.
	maxPeaksPerFrame = 5

	// maxTargetsPerAnchor limits how many target peaks are paired with one anchor peak.
	maxTargetsPerAnchor = 5

	// minDeltaFrames and maxDeltaFrames constrain temporal distance between paired peaks.
	minDeltaFrames = 1
	maxDeltaFrames = 30

	// minPeakBin skips the lowest FFT bins to avoid DC and very low-frequency noise.
	minPeakBin = 5

	// offsetBucketMilliseconds groups nearby offsets into the same vote bucket.
	offsetBucketMilliseconds = 100

	// hanningWindowScale and millisecondsPerSecond make DSP formulas read more explicitly.
	hanningWindowScale    = 0.5
	millisecondsPerSecond = 1000.0

	// hashComponentMask keeps each packed fingerprint component within 16 bits.
	hashComponentMask = 0xFFFF
)
