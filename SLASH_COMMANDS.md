# NFL Discord Bot - Slash Commands & Ephemeral Messages

## 🎯 **New Features Implemented**

### **Slash Commands with Full NFL API Integration**
The bot now supports Discord slash commands with complete NFL API functionality:

- `/help` - Show slash command documentation
- `/stats player:<name> [type:<current|season>] [week:<#>] [year:<year>]` - Player statistics
- `/compare player1:<name> player2:<name> [type:<current|season>] [week:<#>]` - Player comparisons
- `/team team:<name>` - Team information
- `/schedule team:<name>` - Team schedule
- `/scores` - Current week scores

### **Ephemeral Message System**
**Environment Variable: `BOT_VISIBILITY_ROLE`**

**Behavior:**
- If `BOT_VISIBILITY_ROLE` is **not set**: All slash command responses are **public** (everyone can see them)
- If `BOT_VISIBILITY_ROLE` is **set**: All slash command responses are **ephemeral** (only the user who ran the command can see them)

This provides a clean separation:
- **Traditional commands** (`!stats Josh Allen`) → Always public
- **Slash commands** (`/stats player:Josh Allen`) → Ephemeral when visibility role is configured

## 🔧 **Setup Instructions**

### **1. Set Environment Variable (Optional)**
```bash
# For role-restricted visibility (makes slash commands ephemeral)
BOT_VISIBILITY_ROLE="VIP Members"

# Traditional interaction permissions (unchanged)
BOT_ALLOWED_ROLE="Bot Users"
```

### **2. Discord Bot Permissions**
Ensure your Discord application has:
- `applications.commands` scope (for slash commands)
- `bot` scope (for traditional commands)
- All existing permissions (Send Messages, Embed Links, etc.)

### **3. Deploy & Restart**
The bot will automatically:
- Register all slash commands on startup
- Log registration status for each command
- Handle both traditional and slash commands simultaneously

## 🎮 **Usage Examples**

### **Traditional Commands (Always Public)**
```
!stats Josh Allen
!compare Josh Allen vs Mahomes
!team Bills
!schedule Cowboys
!scores
```

### **Slash Commands (Ephemeral when BOT_VISIBILITY_ROLE is set)**
```
/stats player:Josh Allen
/stats player:Josh Allen type:Season
/stats player:Josh Allen week:5
/stats player:Josh Allen week:5 year:2024
/compare player1:Josh Allen player2:Mahomes
/compare player1:Josh Allen player2:Mahomes type:Season
/team team:Bills
/schedule team:Cowboys
/scores
```

## ⚡ **Key Benefits**

### **For Administrators/Moderators**
- **Dual System**: Keep public traditional commands while adding private slash commands
- **Role Control**: Configure who can see slash command responses
- **Clean Chat**: Reduce channel clutter with ephemeral responses

### **For Users**
- **Privacy**: Personal stats queries don't spam the channel
- **Modern UI**: Discord's native slash command interface with autocomplete
- **Same Features**: All existing NFL API functionality available

### **For Server Management**
- **Backward Compatible**: Existing users can continue using `!` commands
- **Flexible**: Can be configured per server needs
- **Professional**: Slash commands provide a more polished bot experience

## 🔍 **Technical Details**

### **Architecture**
- **Hybrid System**: Both message-based and interaction-based handlers
- **Shared Logic**: Same NFL API calls for consistent data
- **Asynchronous**: Uses Discord's followup system for proper slash command handling
- **Role-Based**: Intelligent ephemeral message routing based on configured roles

### **Message Flow**
1. User runs `/stats player:Josh Allen`
2. Bot sends immediate acknowledgment ("⏳ Fetching current week stats...")
3. Bot asynchronously fetches NFL API data
4. Bot sends followup with actual stats embed
5. Response visibility determined by `BOT_VISIBILITY_ROLE` configuration

### **Error Handling**
- Same robust error handling as traditional commands
- Proper ephemeral error messages for failed requests
- API timeout and rate limiting protection maintained

## 🚀 **Production Ready**

The implementation is fully integrated with your existing:
- ✅ Player name fuzzy matching system
- ✅ NFL API client with caching
- ✅ Role-based permissions
- ✅ Bot silence functionality
- ✅ Command acknowledgment system
- ✅ Rich embed formatting
- ✅ Error handling and logging

**Ready for immediate deployment!**
