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
	if RadiumCouchUIConfig.auraAnchor ~= "BuffButton1" then
		-- Older versions stored the center of the invisible BuffFrame manager;
		-- that coordinate is not compatible with the visible button anchor.
		RadiumCouchUIConfig.positions.auras = nil
		RadiumCouchUIConfig.auraAnchor = "BuffButton1"
	end
	return RadiumCouchUIConfig
end

local function Place(frame, point, relative, relativePoint, x, y)
	if not frame then return end
	frame:ClearAllPoints()
	frame:SetPoint(point, relative, relativePoint, x, y)
end

local RuntimeDefaults = {}

local function Print(message)
	DEFAULT_CHAT_FRAME:AddMessage("|cff70d5ffRadium Couch UI:|r " .. message)
end

local function GetAuraAnchor()
	-- In 3.3.5a BuffFrame manages updates but the visible buttons are parented
	-- directly to UIParent. BuffButton1 is the anchor for the visible group.
	return BuffButton1 or BuffFrame
end

local function ApplySavedPosition(id, frame)
	if frame and not RuntimeDefaults[id] then
		local point, relative, relativePoint, x, y = frame:GetPoint()
		RuntimeDefaults[id] = {
			point = point, relative = relative or UIParent,
			relativePoint = relativePoint, x = x, y = y,
		}
	end
	local position = GetConfig().positions[id]
	if not frame or not position then return end
	Place(frame, "CENTER", UIParent, "BOTTOMLEFT", position.x, position.y)
end

function Addon:ApplyUIScale()
	SetCVar("useUiScale", "1")
	SetCVar("uiScale", "1.0")
	UIParent:SetScale(1.04)
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

	-- ConsolePort's real pet ring only exists while a combat pet is out (e.g. a
	-- Death Knight ghoul). The stock row sits on the right face-button cluster
	-- at 1080p, so keep it attached to ConsolePort with a clear lane.
	Place(pet, "TOPRIGHT", bar, "TOPRIGHT", 0, 112)
	ApplySavedPosition("pet", pet)
end


-- MultiBot's playerbot command row is permanently visible by default and lands
-- across the right controller cluster. Keep it hidden and summon it on demand.
Addon.BotMenuToggleName = "RadiumCouchUIBotMenuToggle"
Addon.BotMenuBinding = "CLICK RadiumCouchUIBotMenuToggle:LeftButton"
Addon.PetRingBinding = "CLICK ConsolePortBarPet:MiddleButton"

local function GetBotBar()
	return MultiBot and MultiBot.frames and MultiBot.frames.MultiBar
end

function Addon:PlaceBotMenu()
	local bar = GetBotBar()
	if not bar then return end
	-- The MultiBar anchor is its right edge; its ~530-point row and every
	-- submenu extend left and up from there. Center the row above the
	-- controller clusters without touching MultiBot's saved MultiBarPoint.
	bar:ClearAllPoints()
	bar:SetPoint("BOTTOMRIGHT", UIParent, "BOTTOM", 265, 330)
end

function Addon:SetupBotMenu(dismiss)
	local bar = GetBotBar()
	if not bar or type(MultiBot.ToggleMainUIVisibility) ~= "function" then return end

	if not bar.RadiumCouchUIBotMenu then
		bar:HookScript("OnShow", function() Addon:PlaceBotMenu() end)
		-- Let ConsolePort's interface cursor own the D-pad and face buttons while
		-- the row (or any of its child submenus) is visible.
		if ConsolePort and type(ConsolePort.AddFrame) == "function" then
			ConsolePort:AddFrame(bar)
		end
		bar.RadiumCouchUIBotMenu = true
	end

	-- MultiBot restores its saved visibility at load; the couch layout always
	-- starts with the row dismissed.
	if dismiss and bar:IsShown() then MultiBot.ToggleMainUIVisibility(false) end
end

function Addon:ToggleBotMenu()
	local bar = GetBotBar()
	if not bar or type(MultiBot.ToggleMainUIVisibility) ~= "function" then
		Print("MultiBot is not loaded.")
		return
	end
	self:SetupBotMenu()
	MultiBot.ToggleMainUIVisibility(not bar:IsShown())
end

local BotMenuToggle = CreateFrame("Button", Addon.BotMenuToggleName, UIParent)
BotMenuToggle:SetScript("OnClick", function() Addon:ToggleBotMenu() end)


function Addon:BindControllerChords()
	if not ConsolePort or InCombatLockdown() then return end

	local bindingSet = ConsolePort:GetBindingSet()
	bindingSet.CP_T_R3 = bindingSet.CP_T_R3 or {}
	local chords = bindingSet.CP_T_R3
	local changed = false

	-- LT + R3 toggles the bot command menu. Earlier versions put the pet ring
	-- here; migrate that binding but never overwrite a user's own choice.
	local shift = chords["SHIFT-"]
	if not shift or shift == "" or shift == self.PetRingBinding then
		chords["SHIFT-"] = self.BotMenuBinding
		changed = changed or shift ~= self.BotMenuBinding
	end

	-- RT + R3 holds ConsolePort's pet-command ring when a combat pet is out.
	local ctrl = chords["CTRL-"]
	if not ctrl or ctrl == "" then
		chords["CTRL-"] = self.PetRingBinding
		changed = true
	end

	if not changed then return end
	local data = ConsolePort:GetData()
	data.Bindings = bindingSet
	ConsolePort:LoadBindingSet(bindingSet, true)
end


-- MultiBot (220 degrees), AtlasLoot and DBM (LibDBIcon's 225 default) all pile
-- onto the bottom-left of the minimap where the LFG eye also appears, and the
-- top arc is covered by the wide zone-name banner. Measure where the stock
-- buttons actually sit and give each addon button the nearest free slot to its
-- preferred angle. This runs once per layout version; later drags are kept.
Addon.MinimapLayoutVersion = 2
Addon.MinimapButtonAngles = { DBM = 0, AtlasLoot = 142, MultiBot = 194 }
local MINIMAP_BUTTON_RADIUS = 80
local MINIMAP_OBSTACLES = {
	"MinimapZoneTextButton", "MinimapBorderTop", "MiniMapTracking",
	"MiniMapMailFrame", "MiniMapLFGFrame", "MiniMapBattlefieldFrame",
	"MiniMapWorldMapButton", "MiniMapVoiceChatFrame", "MiniMapInstanceDifficulty",
	"GameTimeFrame", "MinimapZoomIn", "MinimapZoomOut", "TimeManagerClockButton",
}

local function ScreenRect(frame)
	local left, right, top, bottom = frame:GetLeft(), frame:GetRight(), frame:GetTop(), frame:GetBottom()
	if not left or not right or not top or not bottom then return end
	-- Textures (MinimapBorderTop) have no GetEffectiveScale in 3.3.5.
	local owner = frame.GetEffectiveScale and frame or frame:GetParent()
	local scale = owner and owner:GetEffectiveScale() or 1
	return left * scale, right * scale, top * scale, bottom * scale
end

local function Overlaps(a, b)
	return a[1] < b[2] and b[1] < a[2] and a[4] < b[3] and b[4] < a[3]
end

local function FindMinimapAngle(button, preferred, obstacles)
	local mx, my = Minimap:GetCenter()
	if not mx or not my then return preferred end
	local mapScale = Minimap:GetEffectiveScale()
	mx, my = mx * mapScale, my * mapScale
	local half = (button:GetWidth() or 31) * button:GetEffectiveScale() / 2
	local radius = MINIMAP_BUTTON_RADIUS * mapScale

	for step = 0, 60 do
		for _, sign in ipairs(step == 0 and { 1 } or { 1, -1 }) do
			local angle = (preferred + sign * step * 3) % 360
			local x = mx + math.cos(math.rad(angle)) * radius
			local y = my + math.sin(math.rad(angle)) * radius
			local rect = { x - half, x + half, y + half, y - half }
			local free = true
			for _, obstacle in ipairs(obstacles) do
				if Overlaps(rect, obstacle) then free = false break end
			end
			if free then
				obstacles[#obstacles + 1] = rect
				return angle
			end
		end
	end
	return preferred
end

local function PlaceMultiBotMinimapButton()
	local button = _G.MultiBot_MinimapButton
	local config = MultiBot and type(MultiBot.GetMinimapConfig) == "function" and MultiBot.GetMinimapConfig()
	if not button or not config or config.hide then return end
	-- MultiBot scales its radius by the minimap's effective scale even though
	-- the button is already parented to Minimap, pushing it off the rim when
	-- the cluster is enlarged. Use Minimap-local units like LibDBIcon does.
	local angle = math.rad(config.angle or 220)
	button:ClearAllPoints()
	button:SetPoint("CENTER", Minimap, "CENTER",
		math.cos(angle) * MINIMAP_BUTTON_RADIUS, math.sin(angle) * MINIMAP_BUTTON_RADIUS)
end

function Addon:LayoutMinimapButtons()
	local config = GetConfig()
	if config.minimapLayoutVersion ~= self.MinimapLayoutVersion then
		config.minimapButtons = {}
		config.minimapLayoutVersion = self.MinimapLayoutVersion
	end
	local placed = config.minimapButtons

	if MultiBot and type(MultiBot.Minimap_Refresh) == "function" and not self.MultiBotMinimapHooked then
		hooksecurefunc(MultiBot, "Minimap_Refresh", PlaceMultiBotMinimapButton)
		local button = _G.MultiBot_MinimapButton
		if button then button:HookScript("OnDragStop", PlaceMultiBotMinimapButton) end
		self.MultiBotMinimapHooked = true
	end
	PlaceMultiBotMinimapButton()

	-- Every addon button this pass will not move is an obstacle, including
	-- buttons already placed earlier or dragged by the player.
	local icon = LibStub and LibStub("LibDBIcon-1.0", true)
	local pending, obstacles = {}, {}
	for _, name in ipairs(MINIMAP_OBSTACLES) do
		local frame = _G[name]
		local left, right, top, bottom
		if frame then left, right, top, bottom = ScreenRect(frame) end
		if left then obstacles[#obstacles + 1] = { left, right, top, bottom } end
	end
	local function Consider(name, button)
		if not button then return end
		if placed[name] then
			local left, right, top, bottom = ScreenRect(button)
			if left and button:IsShown() then obstacles[#obstacles + 1] = { left, right, top, bottom } end
		else
			pending[#pending + 1] = { name = name, button = button }
		end
	end
	for name, button in pairs(icon and icon.objects or {}) do Consider(name, button) end
	if MultiBot and type(MultiBot.SetMinimapConfig) == "function" then
		Consider("MultiBot", _G.MultiBot_MinimapButton)
	end

	-- Known buttons first, in a stable order, then anything else LibDBIcon owns.
	table.sort(pending, function(a, b)
		local pa, pb = self.MinimapButtonAngles[a.name], self.MinimapButtonAngles[b.name]
		if (pa ~= nil) ~= (pb ~= nil) then return pa ~= nil end
		return a.name < b.name
	end)
	for _, entry in ipairs(pending) do
		local angle = FindMinimapAngle(entry.button, self.MinimapButtonAngles[entry.name] or 220, obstacles)
		if entry.name == "MultiBot" then
			MultiBot.SetMinimapConfig("angle", angle)
			PlaceMultiBotMinimapButton()
		elseif entry.button.db then
			entry.button.db.minimapPos = angle
			icon:Refresh(entry.name, entry.button.db)
		end
		placed[entry.name] = true
	end
end


-- ConsolePort's game menu is anchored to the top of the screen and its System
-- column runs about 25 pixels past the bottom at the RP6 UI scale. Shrink only
-- that menu (ConsolePort's own scale setting) unless the player changed it.
Addon.MenuScale = 0.95

function Addon:LayoutGameMenu()
	local menu = ConsolePortUI_Menu
	local config = ConsolePortUIConfig and ConsolePortUIConfig.Menu
	if not menu or not config or InCombatLockdown() then return end
	if config.scale == nil or config.scale == 1 then config.scale = self.MenuScale end
	menu:SetScale(config.scale)
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

function Addon:LayoutPlayerStatus()
	-- Both frames are owned by Blizzard's position manager. Mark them as
	-- user-placed before applying our saved anchors or the next UI update will
	-- quietly snap them back underneath their editor handles.
	if PlayerFrame then
		PlayerFrame:SetMovable(true)
		PlayerFrame:SetUserPlaced(true)
	end
	local auraAnchor = GetAuraAnchor()
	if auraAnchor then
		auraAnchor:SetMovable(true)
		auraAnchor:SetUserPlaced(true)
	end
	ApplySavedPosition("player", PlayerFrame)
	ApplySavedPosition("auras", auraAnchor)
end


function Addon:SaveEditorMove(handle, x, y, deltaX, deltaY)
	local config = GetConfig()
	if handle.target.id == "immersion" then
		config.immersionX = (ImmersionSetup.boxoffsetX or 0) + deltaX
		config.immersionY = (ImmersionSetup.boxoffsetY or self.DefaultImmersionY) + deltaY
		self:LayoutImmersion()
	else
		config.positions[handle.target.id] = { x = x, y = y }
		ApplySavedPosition(handle.target.id, handle.target.frame)
		if handle.target.id == "chat" and type(FCF_SavePositionAndDimensions) == "function" then
			FCF_SavePositionAndDimensions(ChatFrame1)
		end
	end

	handle:ClearAllPoints()
	handle:SetPoint("CENTER", handle.target.frame, "CENTER")
end

function Addon:SetupLayoutEditor()
	if self.LayoutHandles then return end

	local tracker = WatchFrame or ObjectiveTrackerFrame
	local targets = {
		{ id = "chat", label = "Chat", frame = ChatFrame1 },
		{ id = "minimap", label = "Minimap", frame = MinimapCluster },
		{ id = "objectives", label = "Objectives", frame = tracker },
		{ id = "player", label = "Player Status", frame = PlayerFrame },
		{ id = "auras", label = "Buffs / Status", frame = GetAuraAnchor() },
		{ id = "controller", label = "Controller Bar", frame = ConsolePortBar },
		{ id = "pet", label = "Pet Ring", frame = ConsolePortBarPet },
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

			handle:SetScript("OnDragStop", function(handle)
				handle:StopMovingOrSizing()
				if InCombatLockdown() then return end
				local x, y = handle:GetCenter()
				if not x or not y or not handle.startX or not handle.startY then return end
				Addon:SaveEditorMove(handle, x, y, x - handle.startX, y - handle.startY)
				handle.startX, handle.startY = nil, nil
			end)

			handle:Hide()
			self.LayoutHandles[#self.LayoutHandles + 1] = handle
		end
	end
end

function Addon:SelectEditorTarget(delta)
	local count = #self.LayoutHandles
	self.EditorIndex = ((self.EditorIndex or 1) - 1 + delta) % count + 1
	for index, handle in ipairs(self.LayoutHandles) do
		if index == self.EditorIndex then
			handle:SetBackdropColor(0.55, 0.35, 0.05, 0.96)
			handle:SetBackdropBorderColor(1, 0.82, 0.25, 1)
		else
			handle:SetBackdropColor(0.05, 0.35, 0.55, 0.92)
			handle:SetBackdropBorderColor(0.4, 0.85, 1, 1)
		end
	end
	Print("selected " .. self.LayoutHandles[self.EditorIndex].target.label .. ".")
end

function Addon:MoveSelectedTarget(deltaX, deltaY)
	local handle = self.LayoutHandles and self.LayoutHandles[self.EditorIndex or 1]
	if not handle then return end
	local x, y = handle.target.frame:GetCenter()
	if not x or not y then return end
	self:SaveEditorMove(handle, x + deltaX, y + deltaY, deltaX, deltaY)
end

function Addon:EnableEditorBindings()
	self.EditorInput = self.EditorInput or CreateFrame("Frame", "RadiumCouchUIEditorInput")
	ClearOverrideBindings(self.EditorInput)

	local commands = {
		{ "CP_L_UP", function() Addon:MoveSelectedTarget(0, Addon.EditorFine and 2 or 10) end },
		{ "CP_L_DOWN", function() Addon:MoveSelectedTarget(0, Addon.EditorFine and -2 or -10) end },
		{ "CP_L_LEFT", function() Addon:MoveSelectedTarget(Addon.EditorFine and -2 or -10, 0) end },
		{ "CP_L_RIGHT", function() Addon:MoveSelectedTarget(Addon.EditorFine and 2 or 10, 0) end },
		{ "CP_T1", function() Addon:SelectEditorTarget(-1) end },
		{ "CP_T2", function() Addon:SelectEditorTarget(1) end },
		{ "CP_R_DOWN", function()
			Addon.EditorFine = not Addon.EditorFine
			Print(Addon.EditorFine and "fine movement: 2 points." or "coarse movement: 10 points.")
		end },
		{ "CP_R_RIGHT", function() Addon:LockLayout() end },
	}

	for index, command in ipairs(commands) do
		local button = self.EditorInput[index]
		if not button then
			button = CreateFrame("Button", "RadiumCouchUIEditorButton" .. index, self.EditorInput)
			self.EditorInput[index] = button
		end
		button:SetScript("OnClick", command[2])
		local key = GetBindingKey(command[1])
		if key then SetOverrideBindingClick(self.EditorInput, true, key, button:GetName(), "LeftButton") end
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
	self.EditorIndex = 1
	self.EditorFine = false
	self:SelectEditorTarget(0)
	self:EnableEditorBindings()
	Print("LB/RB select; D-pad move; X/Y-side face button fine/coarse; physical B save and lock.")
end

function Addon:LockLayout()
	if not self.LayoutHandles then return end
	if self.EditorInput and not InCombatLockdown() then ClearOverrideBindings(self.EditorInput) end
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
	for id, position in pairs(RuntimeDefaults) do
		local frame = id == "chat" and ChatFrame1
			or id == "minimap" and MinimapCluster
			or id == "objectives" and (WatchFrame or ObjectiveTrackerFrame)
			or id == "player" and PlayerFrame
			or id == "auras" and GetAuraAnchor()
			or id == "controller" and ConsolePortBar
			or id == "pet" and ConsolePortBarPet
		if frame then Place(frame, position.point, position.relative, position.relativePoint, position.x, position.y) end
	end
	self:LayoutChat()
	self:LayoutNavigation()
	self:LayoutPetBar()
	self:LayoutPlayerStatus()
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
	elseif command == "bots" then
		Addon:ToggleBotMenu()
	else
		Print("commands: /rcui unlock, /rcui lock, /rcui reset, /rcui bots")
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

	self:ApplyUIScale()
	self:LayoutChat()
	self:LayoutNavigation()
	self:LayoutPetBar()
	self:LayoutControllerBar()
	self:LayoutPlayerStatus()
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

if type(UIParent_ManageFramePositions) == "function" then
	hooksecurefunc("UIParent_ManageFramePositions", function()
		if not InCombatLockdown() then Addon:LayoutPlayerStatus() end
	end)
end

if type(GameTooltip_SetDefaultAnchor) == "function" then
	hooksecurefunc("GameTooltip_SetDefaultAnchor", function(tooltip)
		Addon:LayoutTooltip(tooltip)
	end)
end

if type(BuffFrame_UpdateAllBuffAnchors) == "function" then
	hooksecurefunc("BuffFrame_UpdateAllBuffAnchors", function()
		ApplySavedPosition("auras", GetAuraAnchor())
	end)
end

Addon:Hide()

-- Chat dimensions are restored late by FrameXML. Reassert the couch layout
-- once after the world has finished loading, then persist the corrected size.
LateLayout = CreateFrame("Frame", "RadiumCouchUILateLayout")
LateLayout:Hide()
local LateSteps = {
	"ApplyUIScale", "LayoutChat", "LayoutNavigation", "LayoutPetBar",
	"LayoutControllerBar", "LayoutPlayerStatus", "BindControllerChords",
	"SetupBotMenu", "LayoutMinimapButtons", "LayoutGameMenu",
	"SetupImmersionController", "LayoutImmersion",
}

LateLayout:SetScript("OnUpdate", function(self, elapsed)
	self.elapsed = (self.elapsed or 0) + elapsed
	if self.elapsed < 1 then return end
	-- Run once, and isolate each step: Lua errors are usually hidden on the
	-- couch client, so one failure must neither block the rest nor repeat.
	self:Hide()

	for _, step in ipairs(LateSteps) do
		local ok, err = pcall(Addon[step], Addon, step == "SetupBotMenu" or nil)
		if not ok then Print(step .. " failed: " .. tostring(err)) end
	end
	QueueBagLayout()
end)
