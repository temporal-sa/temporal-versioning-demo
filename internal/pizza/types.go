package pizza

// TaskQueue is the single task queue all pizza workers poll.
const TaskQueue = "pizza"

// WorkflowTypeName is the shared workflow type registered by every version, so
// that the type routes across Worker Deployment Versions.
const WorkflowTypeName = "PizzaOrder"

// GetStateQuery is the query name the backend calls to read an order's live state.
const GetStateQuery = "getState"

// StepLabel is the short label shown in the SPA stepper. Values must match the
// labels used by the frontend.
type StepLabel string

// Step labels, ordered by their position in the pipeline.
const (
	StepReceived       StepLabel = "Recv"
	StepCooking        StepLabel = "Cook"
	StepQualityCheck   StepLabel = "QC"
	StepOutForDelivery StepLabel = "Deliv"
	StepDroneDelivery  StepLabel = "Drone"
	StepDelivered      StepLabel = "Done"
)

// Menu is cycled deterministically by order index to name pizzas.
var Menu = []string{
	"Margherita", "Pepperoni", "Quattro Formaggi", "Marinara",
	"Capricciosa", "Diavola", "Hawaiian", "Veggie Supreme",
}

// OrderInput is the workflow argument.
type OrderInput struct {
	OrderID int    `json:"orderId"`
	Pizza   string `json:"pizza"`
}

// OrderState is returned by the getState query and consumed by the backend.
type OrderState struct {
	Version     string      `json:"version"` // "v1" | "v2" | "v3"
	Pizza       string      `json:"pizza"`
	Steps       []StepLabel `json:"steps"`       // full ordered pipeline for this shape
	CurrentStep int         `json:"currentStep"` // index into Steps of the in-progress step
	Done        bool        `json:"done"`        // workflow finished all steps
	Failing     bool        `json:"failing"`     // current step is erroring/retrying
	RetryCount  int         `json:"retryCount"`  // attempts on the failing step
}
