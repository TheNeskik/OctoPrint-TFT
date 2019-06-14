package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gotk3/gotk3/gtk"
	"github.com/mcuadros/go-octoprint"
)

var filamentPanelInstance *filamentPanel

type filamentPanel struct {
	CommonPanel

	amount *StepButton
	tool   *StepButton

	box      *gtk.Box
	labels   map[string]*LabelWithImage
	previous *octoprint.TemperatureState
}

func FilamentPanel(ui *UI, parent Panel) Panel {
	if filamentPanelInstance == nil {
		m := &filamentPanel{CommonPanel: NewCommonPanel(ui, parent),
			labels: map[string]*LabelWithImage{},
		}

		m.b = NewBackgroundTask(time.Second*2, m.updateTemperatures)
		m.initialize()
		filamentPanelInstance = m
	}

	return filamentPanelInstance
}

func (m *filamentPanel) initialize() {
	defer m.Initialize()

	m.Grid().Attach(m.createExtrudeButton("Retracter", "retract.svg", -1), 1, 0, 1, 1)
	m.Grid().Attach(m.createExtrudeButton("Extruder", "extrude.svg", 1), 4, 0, 1, 1)

	m.box = MustBox(gtk.ORIENTATION_VERTICAL, 5)
	m.box.SetVAlign(gtk.ALIGN_CENTER)
	m.box.SetMarginStart(10)

	m.Grid().Attach(m.box, 2, 0, 2, 1)

	m.amount = MustStepButton("move-step.svg", Step{"5mm", 5}, Step{"10mm", 10}, Step{"1mm", 1})
	m.Grid().Attach(m.amount, 2, 1, 1, 1)

	m.Grid().Attach(m.createToolButton(), 1, 1, 1, 1)
	m.Grid().Attach(m.createFlowrateButton(), 3, 1, 1, 1)
	
	m.Grid().Attach(MustButtonImage("Retour", "back.svg", m.UI.GoHistory), 4, 1, 1, 1)
}

func (m *filamentPanel) updateTemperatures() {
	s, err := (&octoprint.ToolStateRequest{
		History: true,
		Limit:   1,
	}).Do(m.UI.Printer)

	if err != nil {
		Logger.Error(err)
		return
	}

	m.loadTemperatureState(s)
}

func (m *filamentPanel) loadTemperatureState(s *octoprint.TemperatureState) {
	for tool, current := range s.Current {
		if _, ok := m.labels[tool]; !ok {
			m.addNewTool(tool)
		}

		m.loadTemperatureData(tool, &current)
	}

	m.previous = s
} 

func (m *filamentPanel) addNewTool(tool string) {
	m.labels[tool] = MustLabelWithImage("extruder.svg", "")
	m.box.Add(m.labels[tool])
	m.tool.AddStep(Step{strings.Title(tool), tool})

	Logger.Infof("New tool detected %s", tool)
}

func (m *filamentPanel) loadTemperatureData(tool string, d *octoprint.TemperatureData) {
	text := fmt.Sprintf("%s: %.1f°C \n⇒ %.1f°C", strings.Title(tool), d.Actual, d.Target)

	if m.previous != nil && d.Target > 0 {
		if p, ok := m.previous.Current[tool]; ok {
			text = fmt.Sprintf("%s\nΔ (%.1f°C)", text, d.Actual-p.Actual)
		}
	}

	m.labels[tool].Label.SetText(text)
	m.labels[tool].ShowAll()
}

func (m *filamentPanel) createToolButton() *StepButton {
	m.tool = MustStepButton("extruder.svg")
	m.tool.Callback = func() {
		cmd := &octoprint.ToolSelectRequest{}
		cmd.Tool = m.tool.Value().(string)

		Logger.Infof("Changing tool to %s", cmd.Tool)
		if err := cmd.Do(m.UI.Printer); err != nil {
			Logger.Error(err)
			return
		}
	}

	return m.tool
}

func (m *filamentPanel) createFlowrateButton() *StepButton {
	b := MustStepButton("speed-step.svg", Step{"Normal", 100}, Step{"Rapide", 125}, Step{"Lent", 75})
	b.Callback = func() {
		cmd := &octoprint.ToolFlowrateRequest{}
		cmd.Factor = b.Value().(int)

		Logger.Infof("Changing flowrate to %d%%", cmd.Factor)
		if err := cmd.Do(m.UI.Printer); err != nil {
			Logger.Error(err)
			return
		}
	}

	return b
}

func (m *filamentPanel) createExtrudeButton(label, image string, dir int) gtk.IWidget {
	return MustButtonImage(label, image, func() {
		cmd := &octoprint.ToolExtrudeRequest{}
		cmd.Amount = m.amount.Value().(int) * dir

		Logger.Infof("Sending extrude request, with amount %d", cmd.Amount)
		if err := cmd.Do(m.UI.Printer); err != nil {
			Logger.Error(err)
			return
		}
	})
}
