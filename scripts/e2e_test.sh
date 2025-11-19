#!/bin/bash
# End-to-End Testing Helper Script
# This script helps automate common E2E testing tasks

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Colors for output
info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[✓]${NC} $1"; }
error() { echo -e "${RED}[✗]${NC} $1"; }
warn() { echo -e "${YELLOW}[⚠]${NC} $1"; }

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check service health
check_service_health() {
    local service=$1
    local port=$2
    local name=$3
    
    if curl -s -f "http://localhost:$port/health" > /dev/null 2>&1; then
        success "$name (port $port) is healthy"
        return 0
    else
        error "$name (port $port) is NOT responding"
        return 1
    fi
}

# Check Redis stream length
check_stream_length() {
    local stream=$1
    local name=$2
    local length=$(docker exec stock-scanner-redis redis-cli XLEN "$stream" 2>/dev/null || echo "0")
    
    if [ "$length" -gt 0 ]; then
        success "$name stream has $length messages"
        return 0
    else
        warn "$name stream is empty (length: $length)"
        return 1
    fi
}

# Check Redis key exists
check_redis_key() {
    local key=$1
    local name=$2
    local exists=$(docker exec stock-scanner-redis redis-cli EXISTS "$key" 2>/dev/null || echo "0")
    
    if [ "$exists" = "1" ]; then
        success "$name key exists: $key"
        return 0
    else
        warn "$name key does not exist: $key"
        return 1
    fi
}

# Main menu
show_menu() {
    echo -e "\n${BLUE}=========================================="
    echo "End-to-End Testing Helper"
    echo "==========================================${NC}\n"
    echo "1. Check all service health"
    echo "2. Check data flow (ticks → bars → indicators)"
    echo "3. Check scanner status"
    echo "4. Check rules in Redis"
    echo "5. Check alerts"
    echo "6. Monitor real-time (watch mode)"
    echo "7. Add test rule to Redis"
    echo "8. View service logs"
    echo "9. Full system check"
    echo "0. Exit"
    echo -e "\n"
}

# Check all services
check_all_services() {
    info "Checking all service health..."
    echo ""
    
    local all_ok=true
    
    check_service_health "ingest" 8081 "Ingest Service" || all_ok=false
    check_service_health "bars" 8083 "Bars Service" || all_ok=false
    check_service_health "indicator" 8085 "Indicator Service" || all_ok=false
    check_service_health "scanner" 8087 "Scanner Service" || all_ok=false
    
    echo ""
    if [ "$all_ok" = true ]; then
        success "All services are healthy!"
    else
        error "Some services are not healthy"
    fi
}

# Check data flow
check_data_flow() {
    info "Checking data flow through the pipeline..."
    echo ""
    
    check_stream_length "ticks" "Ticks"
    check_stream_length "bars.finalized" "Finalized Bars"
    
    # Check indicators
    local indicator_count=$(docker exec stock-scanner-redis redis-cli KEYS "ind:*" 2>/dev/null | wc -l)
    if [ "$indicator_count" -gt 0 ]; then
        success "Found $indicator_count indicator keys"
    else
        warn "No indicator keys found"
    fi
    
    # Check live bars
    local livebar_count=$(docker exec stock-scanner-redis redis-cli KEYS "livebar:*" 2>/dev/null | wc -l)
    if [ "$livebar_count" -gt 0 ]; then
        success "Found $livebar_count live bar keys"
    else
        warn "No live bar keys found"
    fi
}

# Check scanner status
check_scanner_status() {
    info "Checking scanner status..."
    echo ""
    
    if command_exists jq; then
        local stats=$(curl -s http://localhost:8087/stats)
        echo "$stats" | jq '.scan_loop'
        echo ""
        echo "$stats" | jq '.state_manager'
        echo ""
        echo "$stats" | jq '.cooldown_tracker'
    else
        curl -s http://localhost:8087/stats
    fi
}

# Check rules
check_rules() {
    info "Checking rules in Redis..."
    echo ""
    
    local rule_ids=$(docker exec stock-scanner-redis redis-cli SMEMBERS "rules:ids" 2>/dev/null)
    
    if [ -z "$rule_ids" ]; then
        warn "No rules found in Redis"
        echo "Use option 7 to add a test rule"
        return
    fi
    
    success "Found rules:"
    echo "$rule_ids" | while read -r rule_id; do
        if [ -n "$rule_id" ]; then
            echo "  - $rule_id"
            if command_exists jq; then
                docker exec stock-scanner-redis redis-cli GET "rules:$rule_id" 2>/dev/null | jq -r '.name // .id' 2>/dev/null || echo "    (could not parse)"
            fi
        fi
    done
}

# Check alerts
check_alerts() {
    info "Checking alerts..."
    echo ""
    
    local alert_count=$(docker exec stock-scanner-redis redis-cli XLEN alerts 2>/dev/null || echo "0")
    
    if [ "$alert_count" -gt 0 ]; then
        success "Found $alert_count alerts in stream"
        echo ""
        info "Recent alerts:"
        docker exec stock-scanner-redis redis-cli XREAD COUNT 5 STREAMS alerts 0 2>/dev/null | head -20
    else
        warn "No alerts found in stream"
    fi
}

# Monitor real-time
monitor_realtime() {
    info "Starting real-time monitoring (Ctrl+C to stop)..."
    echo ""
    
    watch -n 1 '
        echo "=== Service Health ==="
        curl -s http://localhost:8081/health 2>/dev/null | grep -o "\"status\":\"[^\"]*\"" | head -1 || echo "Ingest: DOWN"
        curl -s http://localhost:8083/health 2>/dev/null | grep -o "\"status\":\"[^\"]*\"" | head -1 || echo "Bars: DOWN"
        curl -s http://localhost:8085/health 2>/dev/null | grep -o "\"status\":\"[^\"]*\"" | head -1 || echo "Indicator: DOWN"
        curl -s http://localhost:8087/health 2>/dev/null | grep -o "\"status\":\"[^\"]*\"" | head -1 || echo "Scanner: DOWN"
        echo ""
        echo "=== Data Flow ==="
        echo "Ticks: $(docker exec stock-scanner-redis redis-cli XLEN ticks 2>/dev/null || echo 0)"
        echo "Bars: $(docker exec stock-scanner-redis redis-cli XLEN bars.finalized 2>/dev/null || echo 0)"
        echo "Indicators: $(docker exec stock-scanner-redis redis-cli KEYS "ind:*" 2>/dev/null | wc -l)"
        echo "Alerts: $(docker exec stock-scanner-redis redis-cli XLEN alerts 2>/dev/null || echo 0)"
        echo ""
        echo "=== Scanner Stats ==="
        curl -s http://localhost:8087/stats 2>/dev/null | grep -o "\"scan_cycles\":[0-9]*" || echo "N/A"
    '
}

# Add test rule
add_test_rule() {
    info "Adding test rule to Redis..."
    echo ""
    
    local rule_id="rule-rsi-oversold-test"
    local rule_json=$(cat <<EOF
{
  "id": "$rule_id",
  "name": "RSI Oversold Test",
  "description": "Test rule: Alert when RSI < 30",
  "conditions": [
    {
      "metric": "rsi_14",
      "operator": "<",
      "value": 30.0
    }
  ],
  "cooldown": 300,
  "enabled": true,
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "updated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
)
    
    echo "$rule_json" | docker exec -i stock-scanner-redis redis-cli SET "rules:$rule_id" "$(cat)" EX 3600
    docker exec stock-scanner-redis redis-cli SADD "rules:ids" "$rule_id"
    
    success "Test rule added: $rule_id"
    echo ""
    info "Rule details:"
    echo "$rule_json" | jq '.' 2>/dev/null || echo "$rule_json"
}

# View logs
view_logs() {
    echo ""
    echo "Select service to view logs:"
    echo "1. Ingest"
    echo "2. Bars"
    echo "3. Indicator"
    echo "4. Scanner"
    echo "5. All services"
    echo ""
    read -p "Choice: " log_choice
    
    case $log_choice in
        1) docker logs -f stock-scanner-ingest ;;
        2) docker logs -f stock-scanner-bars ;;
        3) docker logs -f stock-scanner-indicator ;;
        4) docker logs -f stock-scanner-scanner ;;
        5) docker-compose -f config/docker-compose.yaml logs -f ;;
        *) error "Invalid choice" ;;
    esac
}

# Full system check
full_system_check() {
    info "Running full system check..."
    echo ""
    
    echo -e "${BLUE}=== Infrastructure ===${NC}"
    docker exec stock-scanner-redis redis-cli ping > /dev/null 2>&1 && success "Redis: OK" || error "Redis: FAIL"
    docker exec stock-scanner-timescaledb pg_isready -U postgres > /dev/null 2>&1 && success "TimescaleDB: OK" || error "TimescaleDB: FAIL"
    curl -s -f "http://localhost:9090/-/healthy" > /dev/null 2>&1 && success "Prometheus: OK" || error "Prometheus: FAIL"
    
    echo ""
    echo -e "${BLUE}=== Go Services ===${NC}"
    check_all_services
    
    echo ""
    echo -e "${BLUE}=== Data Flow ===${NC}"
    check_data_flow
    
    echo ""
    echo -e "${BLUE}=== Scanner ===${NC}"
    check_scanner_status
    
    echo ""
    echo -e "${BLUE}=== Rules ===${NC}"
    check_rules
    
    echo ""
    echo -e "${BLUE}=== Alerts ===${NC}"
    check_alerts
}

# Main loop
main() {
    # Check prerequisites
    if ! command_exists docker; then
        error "Docker is not installed"
        exit 1
    fi
    
    if ! command_exists curl; then
        error "curl is not installed"
        exit 1
    fi
    
    # Check if services are running
    if ! docker ps | grep -q stock-scanner-redis; then
        warn "Docker services don't appear to be running"
        echo "Start services with: make docker-up-all"
        echo ""
        read -p "Continue anyway? (y/n) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    
    while true; do
        show_menu
        read -p "Select option: " choice
        echo ""
        
        case $choice in
            1) check_all_services ;;
            2) check_data_flow ;;
            3) check_scanner_status ;;
            4) check_rules ;;
            5) check_alerts ;;
            6) monitor_realtime ;;
            7) add_test_rule ;;
            8) view_logs ;;
            9) full_system_check ;;
            0) info "Exiting..."; exit 0 ;;
            *) error "Invalid option" ;;
        esac
        
        echo ""
        read -p "Press Enter to continue..."
    done
}

# Run main
main

