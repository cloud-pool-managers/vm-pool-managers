package utils

import "PoolManagerVM/backend/models"

// MissingServersByParam retourne, pour chaque Param d’un Serverpool,
// combien de serveurs manquent pour respecter MinVM.
func MissingServersByParam(pool models.Serverpool) []int {
	result := make([]int, len(pool.Params))

	for i, param := range pool.Params {
		count := 0
		for _, s := range pool.ListServ {
			if MatchServerWithParam(s, param) {
				count++
			}
		}

		if param.MinVM <= count+param.PendingJobs {
			result[i] = 0
		} else {
			result[i] = param.MinVM - (count + param.PendingJobs)
		}
	}

	return result
}

// MatchServerWithParam vérifie si un serveur correspond aux paramètres donnés
func MatchServerWithParam(s models.Server, p models.Param) bool {
	if s.ImageRef != p.ImageRef {
		return false
	}
	if s.FlavorRef != p.FlavorRef {
		return false
	}
	if s.ServerpoolID != p.ServerpoolID {
		return false
	}
	if s.UserID != p.UserID {
		return false
	}

	// Vérifier que tous les réseaux de Param sont présents dans le serveur
	if len(p.Networks) > 0 {
		netMap := make(map[string]struct{})
		for _, n := range s.Networks {
			netMap[n] = struct{}{}
		}
		for _, n := range p.Networks {
			if _, ok := netMap[n]; !ok {
				return false
			}
		}
	}

	return true
}
