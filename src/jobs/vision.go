package jobs

type ObjectDetectionCapabilities struct {
	SupportedTypes           []string `json:"supported_media_types"`
	MaxImageSize             int      `json:"max_image_size"`
	MaxImageWidth            int      `json:"max_image_width"`
	MaxImageHeight           int      `json:"max_image_height"`
	MinImageWidth            int      `json:"min_image_width"`
	MinImageHeight           int      `json:"min_image_height"`
	DetectableObjects        []string `json:"detectable_objects"`
	MaxDetectableObjectCount int      `json:"max_detectable_object_num"`
	MinThreshold             float32  `json:"min_threshold"`
	ResponseTypes            []string `json:"response_types"`
	PreferredImageDepth      int      `json:"preferred_image_depth"`
	PreferredImageWidth      int      `json:"preferred_image_width"`
	PreferredImageHeight     int      `json:"preferred_image_height"`
}

func GetObjectDetectionCapabilities() ObjectDetectionCapabilities {
	myCapabilities := ObjectDetectionCapabilities{
		SupportedTypes:           []string{"jpg", "png"},
		MaxImageSize:             1024 * 1024 * 1024,
		MaxImageWidth:            1024,
		MaxImageHeight:           1024,
		MinImageWidth:            128,
		MinImageHeight:           128,
		DetectableObjects:        []string{"person", "bicycle", "car", "motorcycle", "airplane", "bus", "train", "truck", "boat", "traffic light", "fire hydrant", "stop sign", "parking meter", "bench", "bird", "cat", "dog", "horse", "sheep", "cow", "elephant", "bear", "zebra", "giraffe", "backpack", "umbrella", "handbag", "tie", "suitcase", "frisbee", "skis", "snowboard", "sports ball", "kite", "baseball bat", "baseball glove", "skateboard", "surfboard", "tennis racket", "bottle", "wine glass", "cup", "fork", "knife", "spoon", "bowl", "banana", "apple", "sandwich", "orange", "broccoli", "carrot", "hot dog", "pizza", "donut", "cake", "chair", "couch", "potted plant", "bed", "dining table", "toilet", "tv", "laptop", "mouse", "remote", "keyboard", "cell phone", "microwave", "oven", "toaster", "sink", "refrigerator", "book", "clock", "vase", "scissors", "teddy bear", "weapon", "guitar"},
		MaxDetectableObjectCount: 3,
		MinThreshold:             0.5,
		ResponseTypes:            []string{"jpg", "json", "jpg+json"},
		PreferredImageDepth:      24,
		PreferredImageWidth:      640,
		PreferredImageHeight:     640,
	}
	return myCapabilities
}
