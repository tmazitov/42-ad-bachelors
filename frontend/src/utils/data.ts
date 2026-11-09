import { Project } from '@/models/Project'
import { ProjectGroup } from '@/models/ProjectGroup'

export const generalEducationModules = new ProjectGroup('General Education Modules', [
  new Project('GEN 101', 'Success in Disruptive Education', 1, null),
  new Project('GEN 201', 'ICT in 21st Century', 2, null),
  new Project('GEN 301', 'Ethical Practices in AI', 2, null),
])

export const standardModules = new ProjectGroup('Standard Modules', [
  new Project('CCP 101', 'Building a C Standard Library (Libft)', 4, 1314, 'solo'),
  new Project('CCP 111', 'Dynamic Line Reading in C (get_next_line)', 3, 1327, 'solo'),
  new Project('CCP 112', 'Variadic Functions and Formatted Output in C (ft_printf, Exam Rank 2)', 4, 1316, 'solo'),
  new Project('CCP 113', 'Introduction to Linux System Administration (Born to be root)', 2, [1994, 2686], 'solo'),
  new Project('CCP 121', 'Algorithm Optimization and Stack-Based Sorting in C (Push_Swap, Exam Rank 3)', 3, [1471, 2687], 'mixed'),
  new Project('CCP 122', 'Inter-Process Communication in C (Minitalk or Pipex)', 3, [2724, 2005, 2723, 2004], 'solo'),
  new Project('CCP 123', 'Introduction to Computer Graphics in C (FdF or Fractol or SoLong)', 3, [2720, 2008, 2721, 1476, 2722, 2009], 'solo'),
  new Project('CCP 131', 'Concurrent Programming in C (Philosophers)', 4, 1334, 'solo'),
  new Project('CCP 132', 'Building a UNIX Shell in C (Minishell, Exam Rank 4)', 4, [1331, 2727], 'group'),
  new Project('CCP 141', 'Introduction to Networking and IP Addressing (NetPractice)', 3, 2007, 'solo'),
  new Project('CCP 142', 'Fundamentals of 3D Rendering and Raycasting (Cub3d or minirt)', 4, [2725, 1326, 2726, 1315], 'group'),
  new Project('CCP 143', 'C++ Foundations and Object-Oriented Programming (CPP 00 - CPP 04)', 4, [1338, 1339, 1340, 1341, 1342], 'solo'),
  new Project('CCP 151', 'Advanced C++ Programming and STL Mastery (CPP05 - CPP09, Exam Rank 5)', 4, [1343, 1344, 1345, 1346, 2309], 'solo'),
  new Project('CCP 152', 'Infrastructure Virtualization and Secure Web Deployment (Inception)', 4, 1983, 'solo'),
  new Project('CCP 153', 'Network Protocols and Server Implementation (Webserv or IRC, Exam Rank 6)', 4, [1332, 1336], 'group'),
  new Project('CCP 161', 'Full-Stack Web Application & Real-Time Multiplayer Game (ft_transcendence)', 4, 1337, 'group'),
  new Project('ARI 111', 'Python for Data Science', 2, 2267, 'solo'),
  new Project('ARI 121', 'Introduction to Data Science (Piscine Data Science)', 2, 2295, 'solo'),
  new Project('ARI 231', 'Logistic Regression for Data Science (dslr)', 4, 1453, 'group'),
  new Project('ARI 241', 'Multilayer Perceptron and Neural Network (multilayer-perceptron)', 4, 1457, 'solo'),
  new Project('ARI 242', 'Reinforcement Learning with Q-Learning (Learn2Slither)', 4, 2551, 'solo'),
  new Project('ARI 243', 'Brain-Computer Interfaces and EEG Data Processing (total-perspective-vortex)', 4, 1460, 'solo'),
  new Project('ARI 331', 'Machine Learning Algorithms (ft_linear_regression)', 3, 1391, 'solo'),
  new Project('ARI 341', 'Computer Vision and Image-Based Disease Classification (Leaffliction)', 4, 2372, 'group'),
  new Project('MAT 111', 'Polynomial Equation Solving and Mathematical Computation (computorv1)', 3, 1382, 'solo'),
  new Project('MAT 211', 'Linear Algebra Foundations (matrix)', 4, 2077, 'solo'),
  new Project('EMP 101', 'Collaborative Resume (ft_resume)', 2, 2623, 'mixed'),
  new Project('BAI 498', 'Senior Capstone Project', 4, null),
  new Project('BAI 499', 'Internship (Work Experience I)', 4, 1638, 'solo'),
])

export const streamModules = new ProjectGroup('Stream Modules', [
  new Project('GAM 222', '3D Endless Runner Game Development with OpenGL (42run)', 4, 1387, 'solo'),
  new Project('GAM 223', 'Java GUI Development and Object-Oriented Design (swingy)', 3, 1436, 'solo'),
  new Project('GAM 231', 'Modular Game Development (nibbler)', 3, 1386, 'group'),
  new Project('GAM 301', 'Network Programming and Multiplayer Game Development (zappy)', 3, 1463, 'group'),
  new Project('ALG 211', 'Water Flow Simulation and Terrain Modeling (mod1)', 3, 1462, 'group'),
  new Project('GFX 121', 'Hierarchical Modeling and Skeletal Animation (humangl)', 2, 1394, 'group'),
  new Project('KER 221', 'Linux Kernel Programming Challenge (little-penguin-1)', 3, 1416, 'solo'),
  new Project('KER 231', 'Linux Kernel Programming (process-and-memory)', 3, 1421, 'solo'),
  new Project('MOB 121', 'Mobile Contact Management Application (ft_hangouts)', 2, 1379, 'solo'),
  new Project('MOB 122', 'Mobile Application Development with 42 API (swifty-companion)', 2, 1395, 'solo'),
  new Project('MOB 211', 'Collaborative Mobile Application Development (music-room)', 4, 1427, 'group'),
  new Project('MOB 241', 'Mobile Game Development (Peace_Break)', 3, 2552, 'group'),
  new Project('NET 111', 'Network Diagnostics and Packet Analysis (ft_ping)', 2, 1397, 'solo'),
  new Project('NET 121', 'Network Route Tracing and Packet Analysis (ft_traceroute)', 2, 1399, 'solo'),
  new Project('SEC 221', 'Basic Virus Development and Binary Infection (famine)', 3, 1430, 'group'),
  new Project('SEC 222', 'Cybersecurity Challenges (snow-crash)', 4, 1404, 'group'),
  new Project('SYS 111', 'Custom Linux Distribution Development (ft_linux)', 2, 1415, 'solo'),
  new Project('SYS 112', 'Recreating the Unix ls Command (ft_ls)', 2, 1479, 'solo'),
  new Project('SYS 113', 'Low-Level Programming with Assembly Language (libasm)', 2, 1330, 'solo'),
  new Project('SYS 211', 'Job Control and Process Supervision (taskmaster)', 3, 1381, 'group'),
  new Project('SYS 221', 'Compiler Construction - B Language (B)', 2, 2606, 'solo'),
  new Project('WEB 101', 'Web-Based Photo Editing and Social Interaction (camagru)', 2, 1396, 'solo'),
  new Project('WEB 202', 'Full Stack Web Development - Dating Platform (matcha)', 3, 1401, 'group'),
  new Project('WEB 231', 'Web-Based Simulation and Functional Programming (h42n42)', 3, 1429, 'solo'),
  new Project('SEC 211', 'Web Security and Vulnerability Exploitation (darkly)', 3, [1405, 1814], 'group'),
])

export const electiveModules = new ProjectGroup('Elective Modules', [
  new Project('ALG 111', 'Self-Replicating Program Development (dr-quine)', 2, 1418, 'solo'),
  new Project('ALG 212', 'Heuristic Search Algorithms (n-puzzle)', 3, 1385, 'group'),
  new Project('ALG 213', 'Algorithmic Puzzle Solving (rubik)', 3, 1393, 'group'),
  new Project('ALG 221', 'Pathfinding and Optimization Using Graph Algorithms (lem_in)', 3, 1470, 'group'),
  new Project('ALG 222', 'Turing Machines and Computational Theory (ft_turing)', 3, 1403, 'group'),
  new Project('ARI 221', 'Expert Systems and Logical Inference (expert-system)', 3, 1384, 'group'),
  new Project('BLC 211', 'Blockchain Token Development (Tokenizer)', 3, 2485, 'solo'),
  new Project('BLC 221', 'Web3 Development - NFT Creation (TokenizeArt)', 3, 2600, 'solo'),
  new Project('CLD 111', 'Cloud Infrastructure Automation with Ansible (cloud-1)', 3, 1414, 'group'),
  new Project('DEV 111', 'Object-Oriented Programming Piscine (Piscine Object)', 2, 2364, 'solo'),
  new Project('DEV 112', 'Introduction to OCaml (Piscine ocaml)', 2, 2228, 'solo'),
  new Project('DEV 221', 'Advanced C++ Toolbox Development (libftpp)', 3, 2550, 'solo'),
  new Project('DVO 211', 'Introduction to Kubernetes and Cloud-native Application Deployment (Inception-of-Things)', 4, 2064, 'group'),
  new Project('GAM 221', 'Game Engine Development (in-the-shadows)', 2, 1409, 'solo'),
  new Project('MAT 212', 'Introduction to Quantum Programming (Ftl_quantum)', 3, 2225, 'solo'),
  new Project('MAT 213', 'Introduction to Boolean Algebra and Set Theory (ready set boole)', 4, 2076, 'solo'),
  new Project('MAT 221', 'Advanced Computational Mathematics (computorv2)', 4, 1433, 'solo'),
  new Project('MOB 111', 'Introduction to Mobile Apps (Mobile Piscine)', 2, 2355, 'solo'),
  new Project('SEC 111', 'Cybersecurity Fundamentals and Ethical Hacking (Cybersecurity Piscine)', 2, 2346, 'solo'),
  new Project('SYS 223', 'Inter-Process Communication in Multiplayer Games (lem-ipc)', 3, 1464, 'solo'),
])

export const allGroups: ProjectGroup[] = [
  generalEducationModules,
  standardModules,
  streamModules,
  electiveModules,
]

// Streams: a student picks one, and only its modules count toward the stream
// credits (credits from different streams do not add up). Keep in sync with
// backend/internal/domain/curriculum/ad_bachelor.go.
// Modules the student marks completed themselves: their results are recorded
// outside 42 by the Learning Team (on ATRIUM). ARI 111 is here because 42 does
// not mark Python for Data Science validated even when all its modules are done.
// Mirrors adBachelorSelfReported in the backend.
export const SELF_REPORTED = new Set(['GEN 101', 'GEN 201', 'GEN 301', 'ARI 111', 'BAI 498', 'BAI 499'])

// Where students can check those results.
export const ATRIUM_STUDY_PLAN_URL = 'https://atrium.aevsoft.cc/student/study_plan.php'

export interface StreamDef {
  key: string
  name: string
  projects: Project[]
}

function streamProjects(codes: string[]): Project[] {
  return codes.map((code) => {
    const project = streamModules.projects.find((p) => p.code === code)
    if (!project) throw new Error(`stream module ${code} not found`)
    return project
  })
}

export const STREAMS: StreamDef[] = [
  {
    key: 'gaming-graphics',
    name: 'Gaming and Graphics',
    projects: streamProjects(['GAM 222', 'GAM 223', 'GAM 231', 'GAM 301', 'MOB 241', 'ALG 211', 'GFX 121']),
  },
  {
    key: 'mobile-web',
    name: 'Mobile and Web',
    projects: streamProjects(['MOB 121', 'MOB 122', 'MOB 211', 'MOB 241', 'WEB 101', 'WEB 202', 'WEB 231', 'SEC 211']),
  },
  {
    key: 'kernel-systems',
    name: 'Kernel and Systems',
    projects: streamProjects(['KER 221', 'KER 231', 'SYS 111', 'SYS 112', 'SYS 113', 'SYS 211', 'SYS 221']),
  },
  {
    key: 'network-security',
    name: 'Network and Security',
    projects: streamProjects(['NET 111', 'NET 121', 'SEC 221', 'SEC 222', 'SEC 211']),
  },
]
