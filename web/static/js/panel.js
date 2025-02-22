class PanelController {
  constructor() {
    this.baseUrl = this.parsePanelBaseUrl();
    this.initEventListeners();
  }

  parsePanelBaseUrl() {
    const params = new URLSearchParams(window.location.search);
    return `http://${params.get("address")}:${params.get("port")}/${params.get(
      "entrance"
    )}`;
  }

  initEventListeners() {
    document.querySelectorAll(".command-btn").forEach((btn) => {
      btn.addEventListener("click", (e) => this.handleCommand(e));
    });
  }

  async handleCommand(event) {
    const commandGroup = event.target.closest(".command-group");
    const commandType = commandGroup.dataset.command;

    const args = [];
    commandGroup.querySelectorAll(".arg-input").forEach((input) => {
      args.push(input.value);
    });

    try {
      const response = await fetch(`${this.baseUrl}/command`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          command: commandType,
          args: args,
        }),
      });

      const result = await response.json();
      this.displayOutput(result);
    } catch (error) {
      console.error("Command error:", error);
      this.displayOutput({ status: "error", message: "Command failed" });
    }
  }

  displayOutput(result) {
    const output = document.getElementById("commandOutput");
    output.textContent = JSON.stringify(result, null, 2);
    output.style.display = "block";
  }
}

// 初始化
new PanelController();
