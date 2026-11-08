package curriculum

// ADBachelor returns the 42 Abu Dhabi bachelor's curriculum.
// Keep in sync with frontend/src/utils/data.ts until the frontend reads it
// from the API.
func ADBachelor() Curriculum {
	c, err := New(adBachelorModules, adBachelorStreams)
	if err != nil {
		panic(err) // static data; covered by tests
	}
	return c
}

func ids(v ...ProjectID) []ProjectID { return v }

var adBachelorModules = []Module{
	{"GEN 101", "Success in Disruptive Education", 1, General, nil, TeamUnknown},
	{"GEN 201", "ICT in 21st Century", 2, General, nil, TeamUnknown},
	{"GEN 301", "Ethical Practices in AI", 2, General, nil, TeamUnknown},

	{"CCP 101", "Building a C Standard Library (Libft)", 4, Standard, ids(1314), Solo},
	{"CCP 111", "Dynamic Line Reading in C (get_next_line)", 3, Standard, ids(1327), Solo},
	{"CCP 112", "Variadic Functions and Formatted Output in C (ft_printf, Exam Rank 2)", 4, Standard, ids(1316), Solo},
	{"CCP 113", "Introduction to Linux System Administration (Born to be root)", 2, Standard, ids(1994, 2686), Solo},
	{"CCP 121", "Algorithm Optimization and Stack-Based Sorting in C (Push_Swap, Exam Rank 3)", 3, Standard, ids(1471, 2687), Mixed},
	{"CCP 122", "Inter-Process Communication in C (Minitalk or Pipex)", 3, Standard, ids(2724, 2005, 2723, 2004), Solo},
	{"CCP 123", "Introduction to Computer Graphics in C (FdF or Fractol or SoLong)", 3, Standard, ids(2720, 2008, 2721, 1476, 2722, 2009), Solo},
	{"CCP 131", "Concurrent Programming in C (Philosophers)", 4, Standard, ids(1334), Solo},
	{"CCP 132", "Building a UNIX Shell in C (Minishell, Exam Rank 4)", 4, Standard, ids(1331, 2727), Group},
	{"CCP 141", "Introduction to Networking and IP Addressing (NetPractice)", 3, Standard, ids(2007), Solo},
	{"CCP 142", "Fundamentals of 3D Rendering and Raycasting (Cub3d or minirt)", 4, Standard, ids(2725, 1326, 2726, 1315), Group},
	{"CCP 143", "C++ Foundations and Object-Oriented Programming (CPP 00 - CPP 04)", 4, Standard, ids(1338, 1339, 1340, 1341, 1342), Solo},
	{"CCP 151", "Advanced C++ Programming and STL Mastery (CPP05 - CPP09, Exam Rank 5)", 4, Standard, ids(1343, 1344, 1345, 1346, 2309), Solo},
	{"CCP 152", "Infrastructure Virtualization and Secure Web Deployment (Inception)", 4, Standard, ids(1983), Solo},
	{"CCP 153", "Network Protocols and Server Implementation (Webserv or IRC, Exam Rank 6)", 4, Standard, ids(1332, 1336), Group},
	{"CCP 161", "Full-Stack Web Application & Real-Time Multiplayer Game (ft_transcendence)", 4, Standard, ids(1337), Group},
	{"ARI 111", "Python for Data Science", 2, Standard, ids(2267), Solo},
	{"ARI 121", "Introduction to Data Science (Piscine Data Science)", 2, Standard, ids(2295), Solo},
	{"ARI 231", "Logistic Regression for Data Science (dslr)", 4, Standard, ids(1453), Group},
	{"ARI 241", "Multilayer Perceptron and Neural Network (multilayer-perceptron)", 4, Standard, ids(1457), Solo},
	{"ARI 242", "Reinforcement Learning with Q-Learning (Learn2Slither)", 4, Standard, ids(2551), Solo},
	{"ARI 243", "Brain-Computer Interfaces and EEG Data Processing (total-perspective-vortex)", 4, Standard, ids(1460), Solo},
	{"ARI 331", "Machine Learning Algorithms (ft_linear_regression)", 3, Standard, ids(1391), Solo},
	{"ARI 341", "Computer Vision and Image-Based Disease Classification (Leaffliction)", 4, Standard, ids(2372), Group},
	{"MAT 111", "Polynomial Equation Solving and Mathematical Computation (computorv1)", 3, Standard, ids(1382), Solo},
	{"MAT 211", "Linear Algebra Foundations (matrix)", 4, Standard, ids(2077), Solo},
	{"EMP 101", "Collaborative Resume (ft_resume)", 2, Standard, ids(2623), Mixed},
	{"BAI 498", "Senior Capstone Project", 4, Standard, nil, TeamUnknown}, // not tracked through the API
	{"BAI 499", "Internship (Work Experience I)", 4, Standard, ids(1638), Solo},

	{"GAM 222", "3D Endless Runner Game Development with OpenGL (42run)", 4, Stream, ids(1387), Solo},
	{"GAM 223", "Java GUI Development and Object-Oriented Design (swingy)", 3, Stream, ids(1436), Solo},
	{"GAM 231", "Modular Game Development (nibbler)", 3, Stream, ids(1386), Group},
	{"GAM 301", "Network Programming and Multiplayer Game Development (zappy)", 3, Stream, ids(1463), Group},
	{"ALG 211", "Water Flow Simulation and Terrain Modeling (mod1)", 3, Stream, ids(1462), Group},
	{"GFX 121", "Hierarchical Modeling and Skeletal Animation (humangl)", 2, Stream, ids(1394), Group},
	{"KER 221", "Linux Kernel Programming Challenge (little-penguin-1)", 3, Stream, ids(1416), Solo},
	{"KER 231", "Linux Kernel Programming (process-and-memory)", 3, Stream, ids(1421), Solo},
	{"MOB 121", "Mobile Contact Management Application (ft_hangouts)", 2, Stream, ids(1379), Solo},
	{"MOB 122", "Mobile Application Development with 42 API (swifty-companion)", 2, Stream, ids(1395), Solo},
	{"MOB 211", "Collaborative Mobile Application Development (music-room)", 4, Stream, ids(1427), Group},
	{"MOB 241", "Mobile Game Development (Peace_Break)", 3, Stream, ids(2552), Group},
	{"NET 111", "Network Diagnostics and Packet Analysis (ft_ping)", 2, Stream, ids(1397), Solo},
	{"NET 121", "Network Route Tracing and Packet Analysis (ft_traceroute)", 2, Stream, ids(1399), Solo},
	{"SEC 221", "Basic Virus Development and Binary Infection (famine)", 3, Stream, ids(1430), Group},
	{"SEC 222", "Cybersecurity Challenges (snow-crash)", 4, Stream, ids(1404), Group},
	{"SYS 111", "Custom Linux Distribution Development (ft_linux)", 2, Stream, ids(1415), Solo},
	{"SYS 112", "Recreating the Unix ls Command (ft_ls)", 2, Stream, ids(1479), Solo},
	{"SYS 113", "Low-Level Programming with Assembly Language (libasm)", 2, Stream, ids(1330), Solo},
	{"SYS 211", "Job Control and Process Supervision (taskmaster)", 3, Stream, ids(1381), Group},
	{"SYS 221", "Compiler Construction - B Language (B)", 2, Stream, ids(2606), Solo},
	{"WEB 101", "Web-Based Photo Editing and Social Interaction (camagru)", 2, Stream, ids(1396), Solo},
	{"WEB 202", "Full Stack Web Development - Dating Platform (matcha)", 3, Stream, ids(1401), Group},
	{"WEB 231", "Web-Based Simulation and Functional Programming (h42n42)", 3, Stream, ids(1429), Solo},
	{"SEC 211", "Web Security and Vulnerability Exploitation (darkly)", 3, Stream, ids(1405, 1814), Group},

	{"ALG 111", "Self-Replicating Program Development (dr-quine)", 2, Elective, ids(1418), Solo},
	{"ALG 212", "Heuristic Search Algorithms (n-puzzle)", 3, Elective, ids(1385), Group},
	{"ALG 213", "Algorithmic Puzzle Solving (rubik)", 3, Elective, ids(1393), Group},
	{"ALG 221", "Pathfinding and Optimization Using Graph Algorithms (lem_in)", 3, Elective, ids(1470), Group},
	{"ALG 222", "Turing Machines and Computational Theory (ft_turing)", 3, Elective, ids(1403), Group},
	{"ARI 221", "Expert Systems and Logical Inference (expert-system)", 3, Elective, ids(1384), Group},
	{"BLC 211", "Blockchain Token Development (Tokenizer)", 3, Elective, ids(2485), Solo},
	{"BLC 221", "Web3 Development - NFT Creation (TokenizeArt)", 3, Elective, ids(2600), Solo},
	{"CLD 111", "Cloud Infrastructure Automation with Ansible (cloud-1)", 3, Elective, ids(1414), Group},
	{"DEV 111", "Object-Oriented Programming Piscine (Piscine Object)", 2, Elective, ids(2364), Solo},
	{"DEV 112", "Introduction to OCaml (Piscine ocaml)", 2, Elective, ids(2228), Solo},
	{"DEV 221", "Advanced C++ Toolbox Development (libftpp)", 3, Elective, ids(2550), Solo},
	{"DVO 211", "Introduction to Kubernetes and Cloud-native Application Deployment (Inception-of-Things)", 4, Elective, ids(2064), Group},
	{"GAM 221", "Game Engine Development (in-the-shadows)", 2, Elective, ids(1409), Solo},
	{"MAT 212", "Introduction to Quantum Programming (Ftl_quantum)", 3, Elective, ids(2225), Solo},
	{"MAT 213", "Introduction to Boolean Algebra and Set Theory (ready set boole)", 4, Elective, ids(2076), Solo},
	{"MAT 221", "Advanced Computational Mathematics (computorv2)", 4, Elective, ids(1433), Solo},
	{"MOB 111", "Introduction to Mobile Apps (Mobile Piscine)", 2, Elective, ids(2355), Solo},
	{"SEC 111", "Cybersecurity Fundamentals and Ethical Hacking (Cybersecurity Piscine)", 2, Elective, ids(2346), Solo},
	{"SYS 223", "Inter-Process Communication in Multiplayer Games (lem-ipc)", 3, Elective, ids(1464), Solo},
}

// adBachelorStreams are the stream choices (from the curriculum sheet).
var adBachelorStreams = []StreamChoice{
	{"gaming-graphics", "Gaming and Graphics",
		[]string{"GAM 222", "GAM 223", "GAM 231", "GAM 301", "MOB 241", "ALG 211", "GFX 121"}},
	{"mobile-web", "Mobile and Web",
		[]string{"MOB 121", "MOB 122", "MOB 211", "MOB 241", "WEB 101", "WEB 202", "WEB 231", "SEC 211"}},
	{"kernel-systems", "Kernel and Systems",
		[]string{"KER 221", "KER 231", "SYS 111", "SYS 112", "SYS 113", "SYS 211", "SYS 221"}},
	{"network-security", "Network and Security",
		[]string{"NET 111", "NET 121", "SEC 221", "SEC 222", "SEC 211"}},
}
