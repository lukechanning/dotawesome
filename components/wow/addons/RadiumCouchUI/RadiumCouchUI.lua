local Addon = CreateFrame("Frame", "RadiumCouchUI")
local LateLayout

-- Keep the controller clusters and action display unobstructed. Values are UI
-- points, so they follow the client's configured UI scale.
Addon.ReserveBottom = 210
Addon.ReserveTopRight = 255
Addon.Gap = 8
Addon.DefaultImmersionY = 205

local function GetConfig()
	RadiumCouchUIConfig = RadiumCouchUIConfig or {}
	RadiumCouchUIConfig.positions = RadiumCouchUIConfig.positions or {}
	return RadiumCouchUIConfig
end

local function Place(frame, point, relative, relativePoint, x, y)
	if not frame then return end
	frame:ClearAllPoints()
	frame:SetPoint(point, relative, relativePoint, x, y)
end

local function ApplySavedPosition(id, frame)
	local position = GetConfig().positions[id]
	if not frame or not position then return end
	Place(frame, "CENTER", UIParent, "BOTTOMLEFT", position.x, position.y)
end

function Addon:LayoutChat()
	local chat = ChatFrame1
	if not chat then return end

	FCF_SetLocked(chat, 1)
	chat:SetClampedToScreen(true)
	chat:SetMinResize(360, 130)
	chat:SetMaxResize(560, 220)
	if type(FCF_SetWindowSize) == "function" then
		FCF_SetWindowSize(chat, 510, 185)
	else
		chat:SetWidth(510)
		chat:SetHeight(185)
	end
	Place(chat, "TOPLEFT", UIParent, "TOPLEFT", 28, -34)
	ApplySavedPosition("chat", chat)

	local font, _, flags = chat:GetFont()
	chat:SetFont(font or STANDARD_TEXT_FONT, 14, flags or "")
	if type(FCF_SavePositionAndDimensions) == "function" then
		FCF_SavePositionAndDimensions(chat)
	end
end


function Addon:LayoutNavigation()
	if MinimapCluster then
		MinimapCluster:SetScale(1.18)
		Place(MinimapCluster, "TOPRIGHT", UIParent, "TOPRIGHT", -18, -18)
		ApplySavedPosition("minimap", MinimapCluster)
	end

	-- WotLK uses WatchFrame; ObjectiveTrackerFrame keeps this compatible with
	-- clients carrying the newer tracker backport.
	local tracker = WatchFrame or ObjectiveTrackerFrame
	if tracker then
		tracker:SetScale(1.12)
		tracker:SetWidth(300)
		Place(tracker, "TOPRIGHT", UIParent, "TOPRIGHT", -26, -255)
		ApplySavedPosition("objectives", tracker)
	end
end


function Addon:LayoutPetBar()
	local bar = ConsolePortBar
	local pet = ConsolePortBarPet
	if not bar or not pet or InCombatLockdown() then return end

	-- The stock pet-command row sits directly on the right face-button cluster
	-- at 1080p. Keep it attached to ConsolePort, just give it a clear lane.
	Place(pet, "TOPRIGHT", bar, "TOPRIGHT", 0, 112)
	ApplySavedPosition("pet", pet)
end


function Addon:BindPetRing()
	if not ConsolePort or InCombatLockdown() then return end

	local bindingSet = ConsolePort:GetBindingSet()
	bindingSet.CP_T_R3 = bindingSet.CP_T_R3 or {}
	local current = bindingSet.CP_T_R3["SHIFT-"]
	if current and current ~= "" then return end

	bindingSet.CP_T_R3["SHIFT-"] = "CLICK ConsolePortBarPet:MiddleButton"
	local data = ConsolePort:GetData()
	data.Bindings = bindingSet
	ConsolePort:LoadBindingSet(bindingSet, true)
end


function Addon:SetupImmersionController()
	local immersion = ImmersionFrame
	if not immersion or immersion.RadiumControllerAdapter then return end
	if type(immersion.ParseControllerCommand) ~= "function" then return end

	local input = CreateFrame("Frame", "RadiumCouchUIImmersionInput")
	local bindings = {
		"CP_R_DOWN", "CP_R_RIGHT", "CP_R_LEFT", "CP_R_UP",
		"CP_L_DOWN", "CP_L_RIGHT", "CP_L_LEFT", "CP_L_UP",
	}

	for index, binding in ipairs(bindings) do
		local button = CreateFrame("Button", "RadiumCouchUIImmersionButton" .. index, input)
		button.binding = binding
		button:SetScript("OnClick", function(self)
			local key = GetBindingKey(self.binding)
			if key then immersion:ParseControllerCommand(key) end
		end)
		input[index] = button
	end

	local function EnableController()
		if InCombatLockdown() then return end
		ClearOverrideBindings(input)
		immersion:SetImmersionFocus()
		for index, button in ipairs(input) do
			local key = GetBindingKey(button.binding)
			if key then
				SetOverrideBindingClick(input, true, key, button:GetName(), "LeftButton")
			end
		end
	end

	local function DisableController()
		if not InCombatLockdown() then ClearOverrideBindings(input) end
		immersion:ClearImmersionFocus()
	end

	immersion:HookScript("OnShow", EnableController)
	immersion:HookScript("OnHide", DisableController)
	immersion.RadiumControllerAdapter = true
	if immersion:IsShown() then EnableController() end
end


function Addon:LayoutImmersion()
	if not ImmersionSetup or not ImmersionFrame then return end
	local config = GetConfig()

	ImmersionSetup.boxpoint = "Bottom"
	ImmersionSetup.boxoffsetX = config.immersionX or 0
	ImmersionSetup.boxoffsetY = config.immersionY or self.DefaultImmersionY

	local talkbox = ImmersionFrame.TalkBox
	if talkbox and type(talkbox.SetOffset) == "function" then
		talkbox:SetOffset(ImmersionSetup.boxoffsetX, ImmersionSetup.boxoffsetY)
	end
end


local layingOutBags = false

function Addon:LayoutBags()
	if layingOutBags then return end
	layingOutBags = true

	local x = -18
	local y = self.ReserveBottom
	local columnWidth = 0
	local top = UIParent:GetHeight() - self.ReserveTopRight

	for index = 1, (NUM_CONTAINER_FRAMES or 13) do
		local bag = _G["ContainerFrame" .. index]
		if bag and bag:IsShown() then
			local width = bag:GetWidth() or 192
			local height = bag:GetHeight() or 200

			if y > self.ReserveBottom and y + height > top then
				x = x - columnWidth - self.Gap
				y = self.ReserveBottom
				columnWidth = 0
			end

			Place(bag, "BOTTOMRIGHT", UIParent, "BOTTOMRIGHT", x, y)
			y = y + height + self.Gap
			if width > columnWidth then columnWidth = width end
		end
	end

	layingOutBags = false
end


function Addon:LayoutTooltip(tooltip)
	if not tooltip then return end
	tooltip:ClearAllPoints()
	tooltip:SetPoint("TOP", UIParent, "TOP", 0, -32)
end


function Addon:LayoutControllerBar()
	ApplySavedPosition("controller", ConsolePortBar)
end


local function Print(message)
	DEFAULT_CHAT_FRAME:AddMessage("|cff70d5ffRadium Couch UI:|r " .. message)
end

function Addon:SetupLayoutEditor()
	if self.LayoutHandles then return end

	local tracker = WatchFrame or ObjectiveTrackerFrame
	local targets = {
		{ id = "chat", label = "Chat", frame = ChatFrame1 },
		{ id = "minimap", label = "Minimap", frame = MinimapCluster },
		{ id = "objectives", label = "Objectives", frame = tracker },
		{ id = "controller", label = "Controller Bar", frame = ConsolePortBar },
		{ id = "pet", label = "Pet Bar", frame = ConsolePortBarPet },
		{ id = "immersion", label = "Immersion", frame = ImmersionFrame and ImmersionFrame.TalkBox },
	}

	self.LayoutHandles = {}
	for _, target in ipairs(targets) do
		if target.frame then
			local handle = CreateFrame("Button", nil, UIParent)
			handle:SetSize(132, 26)
			handle:SetFrameStrata("TOOLTIP")
			handle:SetMovable(true)
			handle:RegisterForDrag("LeftButton")
			handle:SetBackdrop({
				bgFile = "Interface\\Tooltips\\UI-Tooltip-Background",
				edgeFile = "Interface\\Tooltips\\UI-Tooltip-Border",
				tile = true, tileSize = 16, edgeSize = 12,
				insets = { left = 3, right = 3, top = 3, bottom = 3 },
			})
			handle:SetBackdropColor(0.05, 0.35, 0.55, 0.92)
			handle:SetBackdropBorderColor(0.4, 0.85, 1, 1)
			handle.text = handle:CreateFontString(nil, "OVERLAY", "GameFontNormal")
			handle.text:SetPoint("CENTER")
			handle.text:SetText(target.label)
			handle.target = target

			handle:SetScript("OnDragStart", function(self)
				if InCombatLockdown() then return end
				local x, y = self:GetCenter()
				if not x or not y then return end
				self.startX, self.startY = x, y
				self:ClearAllPoints()
				self:SetPoint("CENTER", UIParent, "BOTTOMLEFT", x, y)
				self:StartMoving()
			end)

			handle:SetScript("OnDragStop", function(self)
				self:StopMovingOrSizing()
				if InCombatLockdown() then return end
				local x, y = self:GetCenter()
				if not x or not y or not self.startX or not self.startY then return end

				local config = GetConfig()
				if self.target.id == "immersion" then
					config.immersionX = (ImmersionSetup.boxoffsetX or 0) + x - self.startX
					config.immersionY = (ImmersionSetup.boxoffsetY or Addon.DefaultImmersionY) + y - self.startY
					Addon:LayoutImmersion()
				else
					config.positions[self.target.id] = { x = x, y = y }
					ApplySavedPosition(self.target.id, self.target.frame)
					if self.target.id == "chat" and type(FCF_SavePositionAndDimensions) == "function" then
						FCF_SavePositionAndDimensions(ChatFrame1)
					end
				end

				self.startX, self.startY = nil, nil
				self:ClearAllPoints()
				self:SetPoint("CENTER", self.target.frame, "CENTER")
			end)

			handle:Hide()
			self.LayoutHandles[#self.LayoutHandles + 1] = handle
		end
	end
end

function Addon:UnlockLayout()
	if InCombatLockdown() then
		Print("layout cannot be unlocked during combat.")
		return
	end
	self:SetupLayoutEditor()
	for _, handle in ipairs(self.LayoutHandles) do
		handle:ClearAllPoints()
		handle:SetPoint("CENTER", handle.target.frame, "CENTER")
		handle:Show()
	end
	Print("unlocked; drag the blue labels, then run /rcui lock.")
end

function Addon:LockLayout()
	if not self.LayoutHandles then return end
	for _, handle in ipairs(self.LayoutHandles) do handle:Hide() end
	Print("locked and saved.")
end

function Addon:ResetLayout()
	if InCombatLockdown() then
		Print("layout cannot be reset during combat.")
		return
	end
	local config = GetConfig()
	config.positions = {}
	config.immersionX = nil
	config.immersionY = nil
	if ConsolePortBar then Place(ConsolePortBar, "BOTTOM", UIParent, "BOTTOM", 0, 0) end
	self:LayoutChat()
	self:LayoutNavigation()
	self:LayoutPetBar()
	self:LayoutImmersion()
	self:LockLayout()
	Print("RP6 defaults restored.")
end

SLASH_RADIUMCOUCHUI1 = "/rcui"
SlashCmdList.RADIUMCOUCHUI = function(message)
	local command = string.lower((message or ""):match("^%s*(%S*)") or "")
	if command == "unlock" then
		Addon:UnlockLayout()
	elseif command == "lock" then
		Addon:LockLayout()
	elseif command == "reset" then
		Addon:ResetLayout()
	else
		Print("commands: /rcui unlock, /rcui lock, /rcui reset")
	end
end


local pendingBags = false
local function QueueBagLayout()
	pendingBags = true
	Addon:Show()
end

Addon:SetScript("OnUpdate", function(self)
	if pendingBags then
		pendingBags = false
		self:LayoutBags()
	end
	self:Hide()
end)

Addon:SetScript("OnEvent", function(self, event, name)
	if event == "ADDON_LOADED" and name ~= "RadiumCouchUI" then return end

	self:LayoutChat()
	self:LayoutNavigation()
	self:LayoutPetBar()
	self:LayoutControllerBar()
	QueueBagLayout()
	LateLayout.elapsed = 0
	LateLayout:Show()
end)

Addon:RegisterEvent("ADDON_LOADED")
Addon:RegisterEvent("PLAYER_ENTERING_WORLD")

for index = 1, (NUM_CONTAINER_FRAMES or 13) do
	local bag = _G["ContainerFrame" .. index]
	if bag then bag:HookScript("OnShow", QueueBagLayout) end
end

if type(ContainerFrame_UpdateAll) == "function" then
	hooksecurefunc("ContainerFrame_UpdateAll", QueueBagLayout)
end

if type(GameTooltip_SetDefaultAnchor) == "function" then
	hooksecurefunc("GameTooltip_SetDefaultAnchor", function(tooltip)
		Addon:LayoutTooltip(tooltip)
	end)
end

Addon:Hide()

-- Chat dimensions are restored late by FrameXML. Reassert the couch layout
-- once after the world has finished loading, then persist the corrected size.
LateLayout = CreateFrame("Frame", "RadiumCouchUILateLayout")
LateLayout:Hide()
LateLayout:SetScript("OnUpdate", function(self, elapsed)
	self.elapsed = (self.elapsed or 0) + elapsed
	if self.elapsed < 1 then return end

	Addon:LayoutChat()
	Addon:LayoutNavigation()
	Addon:LayoutPetBar()
	Addon:LayoutControllerBar()
	Addon:BindPetRing()
	Addon:SetupImmersionController()
	Addon:LayoutImmersion()
	QueueBagLayout()
	self:Hide()
end)
