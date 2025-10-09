[1mdiff --git a/api.go b/api.go[m
[1mindex bfe07bf..9890ee2 100644[m
[1m--- a/api.go[m
[1m+++ b/api.go[m
[36m@@ -5469,14 +5469,14 @@[m [mfunc Handler(w http.ResponseWriter, r *http.Request) {[m
 			// Generate optimized title using AI (mock for now)[m
 			originalTitle := title.String[m
 			optimizedTitle := fmt.Sprintf("%s | Premium Quality | Fast Shipping", originalTitle)[m
[31m-			[m
[32m+[m
 			// Calculate mock scores[m
 			score := 85[m
 			improvement := 15.5[m
 [m
 			// Save optimization history[m
 			historyID := fmt.Sprintf("%d", time.Now().UnixNano())[m
[31m-			[m
[32m+[m
 			c.JSON(http.StatusOK, gin.H{[m
 				"optimization_id":   historyID,[m
 				"product_id":        productID,[m
