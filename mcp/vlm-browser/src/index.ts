import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from "@modelcontextprotocol/sdk/types.js";
import { chromium, Browser, Page } from "playwright";

let browser: Browser | null = null;
let page: Page | null = null;
let markMap: Record<string, { x: number, y: number }> = {};

const server = new Server(
  {
    name: "vlm-browser-mcp",
    version: "1.0.0",
  },
  {
    capabilities: {
      tools: {},
    },
  }
);

async function initBrowser() {
  if (!browser) {
    browser = await chromium.launch({ headless: true });
    page = await browser.newPage();
  }
  return page;
}

async function annotateAndScreenshot(p: Page) {
  // Set-of-Mark (SOM) DOM Injection
  const marks = await p.evaluate(() => {
    // Remove existing marks
    document.querySelectorAll('.vlm-som-mark').forEach(el => el.remove());

    const elements = document.querySelectorAll('a, button, input, select, textarea, [role="button"]');
    const marks: Record<string, { x: number, y: number }> = {};
    
    let counter = 0;
    elements.forEach((el) => {
      const rect = el.getBoundingClientRect();
      if (rect.width > 0 && rect.height > 0) {
        const id = 'A' + counter;
        counter++;
        marks[id] = { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 };
        
        const badge = document.createElement('div');
        badge.className = 'vlm-som-mark';
        badge.style.position = 'absolute';
        badge.style.left = rect.left + window.scrollX + 'px';
        badge.style.top = rect.top + window.scrollY + 'px';
        badge.style.backgroundColor = 'red';
        badge.style.color = 'white';
        badge.style.fontSize = '10px';
        badge.style.fontWeight = 'bold';
        badge.style.padding = '1px 3px';
        badge.style.zIndex = '999999';
        badge.innerText = id;
        
        const border = document.createElement('div');
        border.className = 'vlm-som-mark';
        border.style.position = 'absolute';
        border.style.left = rect.left + window.scrollX + 'px';
        border.style.top = rect.top + window.scrollY + 'px';
        border.style.width = rect.width + 'px';
        border.style.height = rect.height + 'px';
        border.style.border = '2px solid red';
        border.style.pointerEvents = 'none';
        border.style.zIndex = '999998';
        
        document.body.appendChild(badge);
        document.body.appendChild(border);
      }
    });
    return marks;
  });

  markMap = marks;
  const screenshot = await p.screenshot({ type: "jpeg", quality: 60 });
  return screenshot;
}

server.setRequestHandler(ListToolsRequestSchema, async () => {
  return {
    tools: [
      {
        name: "navigate",
        description: "Navigate to a URL",
        inputSchema: {
          type: "object",
          properties: {
            url: { type: "string" },
          },
          required: ["url"],
        },
      },
      {
        name: "click",
        description: "Click a mark ID or x, y coordinate",
        inputSchema: {
          type: "object",
          properties: {
            mark_id: { type: "string" },
            x: { type: "number" },
            y: { type: "number" },
          },
        },
      },
      {
        name: "type_text",
        description: "Type text into a mark ID or coordinate",
        inputSchema: {
          type: "object",
          properties: {
            mark_id: { type: "string" },
            x: { type: "number" },
            y: { type: "number" },
            text: { type: "string" },
          },
          required: ["text"],
        },
      },
      {
        name: "scroll",
        description: "Scroll the page up or down",
        inputSchema: {
          type: "object",
          properties: {
            direction: { type: "string", enum: ["up", "down"] },
          },
          required: ["direction"],
        },
      },
      {
        name: "get_screenshot",
        description: "Get the current screenshot in base64",
        inputSchema: {
          type: "object",
          properties: {},
        },
      },
    ],
  };
});

server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const p = await initBrowser();

  if (request.params.name === "navigate") {
    const { url } = request.params.arguments as any;
    await p!.goto(url);
    const screenshot = await annotateAndScreenshot(p!);
    return {
      content: [
        { type: "text", text: "Navigated to " + url },
        { type: "image", data: screenshot.toString("base64"), mimeType: "image/jpeg" }
      ],
    };
  }
  
  if (request.params.name === "click") {
    const { mark_id, x, y } = request.params.arguments as any;
    if (mark_id && markMap[mark_id]) {
       await p!.mouse.click(markMap[mark_id].x, markMap[mark_id].y);
    } else if (x !== undefined && y !== undefined) {
       await p!.mouse.click(x, y);
    }
    const screenshot = await annotateAndScreenshot(p!);
    return {
      content: [
        { type: "text", text: "Clicked element" },
        { type: "image", data: screenshot.toString("base64"), mimeType: "image/jpeg" }
      ],
    };
  }

  if (request.params.name === "type_text") {
    const { mark_id, x, y, text } = request.params.arguments as any;
    if (mark_id && markMap[mark_id]) {
       await p!.mouse.click(markMap[mark_id].x, markMap[mark_id].y);
    } else if (x !== undefined && y !== undefined) {
       await p!.mouse.click(x, y);
    }
    await p!.keyboard.type(text);
    const screenshot = await annotateAndScreenshot(p!);
    return {
      content: [
        { type: "text", text: "Typed text" },
        { type: "image", data: screenshot.toString("base64"), mimeType: "image/jpeg" }
      ],
    };
  }

  if (request.params.name === "scroll") {
    const { direction } = request.params.arguments as any;
    if (direction === "up") {
      await p!.mouse.wheel(0, -500);
    } else {
      await p!.mouse.wheel(0, 500);
    }
    await p!.waitForTimeout(200);
    const screenshot = await annotateAndScreenshot(p!);
    return {
      content: [
        { type: "text", text: `Scrolled ${direction}` },
        { type: "image", data: screenshot.toString("base64"), mimeType: "image/jpeg" }
      ],
    };
  }

  if (request.params.name === "get_screenshot") {
    const screenshot = await annotateAndScreenshot(p!);
    return {
      content: [
        { type: "text", text: "Screenshot captured" },
        { type: "image", data: screenshot.toString("base64"), mimeType: "image/jpeg" }
      ],
    };
  }

  throw new Error("Unknown tool: " + request.params.name);
});

async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
}

main().catch(console.error);
